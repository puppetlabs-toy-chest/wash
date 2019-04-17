package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/puppetlabs/wash/activity"
)

// FileCacheProvider is a credentials.Provider implementation that wraps an underlying Provider
// (contained in credentials.Credentials) and provides caching support for credentials for the
// specified profile.
type FileCacheProvider struct {
	credentials      *credentials.Credentials
	profile          string
	cachedCredential cachedCredential
}

// NewFileCacheProvider creates a new Provider implementation that wraps a provided Credentials,
// and works with an on disk cache to speed up credential usage when the cached copy is not expired.
// If there are any problems accessing or initializing the cache, an error will be returned, and
// callers should just use the existing credentials provider.
func newFileCacheProvider(ctx context.Context, profile string, creds *credentials.Credentials) (FileCacheProvider, error) {
	if creds == nil {
		return FileCacheProvider{}, errors.New("no underlying Credentials object provided")
	}

	filename, err := cacheFilename(profile)
	if err != nil {
		return FileCacheProvider{}, err
	}

	// load credential if it exists.
	var cachedCredential cachedCredential
	if info, err := os.Stat(filename); !os.IsNotExist(err) {
		if info.Mode()&0077 != 0 {
			// cache file has secret credentials and should only be accessible to the user, refuse to use it.
			return FileCacheProvider{}, fmt.Errorf("cache file %s is not private, please ensure only current user has access", filename)
		}

		activity.Record(ctx, "Loading cached credentials from %v", filename)
		cachedCredential, err = readCache(filename)
		if err != nil {
			// can't read or parse cache, refuse to use it.
			return FileCacheProvider{}, err
		}
	}

	return FileCacheProvider{creds, profile, cachedCredential}, nil
}

// Retrieve implements the Provider interface, returning the cached credential if not expired,
// otherwise fetching the credential from the underlying Provider and caching the results on disk
// with an expiration time.
func (f *FileCacheProvider) Retrieve() (credentials.Value, error) {
	if !f.cachedCredential.IsExpired() {
		// use the cached credential
		return f.cachedCredential.Credential, nil
	}

	// fetch the credentials from the underlying Provider
	credential, err := f.credentials.Get()
	if err != nil {
		return credential, err
	}

	expiration, err := f.credentials.ExpiresAt()
	if err != nil {
		// Fallback to the original credential
		return credential, nil
	}

	// underlying provider supports Expirer interface, so we can cache
	filename, err := cacheFilename(f.profile)
	if err != nil {
		activity.Record(context.Background(), "Unable to determine cache location for %s: %v", f.profile, err)
		return credential, nil
	}

	// update cached credential and save to disk
	f.cachedCredential = cachedCredential{credential, expiration}
	if err = writeCache(filename, f.cachedCredential); err != nil {
		activity.Record(context.Background(), "Unable to update credential cache %s: %v", filename, err)
	}
	return credential, err
}

// IsExpired implements the Provider interface, deferring to the cached credential first,
// but fall back to the underlying Provider if it is expired.
func (f *FileCacheProvider) IsExpired() bool {
	return f.cachedCredential.IsExpired() && f.credentials.IsExpired()
}

// ExpiresAt implements the Expirer interface, and gives access to the expiration time of the credential
func (f *FileCacheProvider) ExpiresAt() time.Time {
	return f.cachedCredential.Expiration
}

// cacheFilename returns the name of the credential cache file for requested profile. It also
// makes sure the directory containing the cache file exists.
func cacheFilename(profile string) (string, error) {
	cdir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	cachedir := filepath.Join(cdir, "wash", "aws-credentials")

	// ensure cache directory exists
	if err = os.MkdirAll(cachedir, 0750); err != nil {
		return "", err
	}

	return filepath.Join(cachedir, profile+".json"), nil
}

// cachedCredential is a single cached credential, along with expiration time
type cachedCredential struct {
	Credential credentials.Value
	Expiration time.Time
}

// IsExpired determines if the cached credential has expired
func (c *cachedCredential) IsExpired() bool {
	return c.Expiration.Before(time.Now())
}

// readCache reads the contents of the credential cache and returns the
// parsed json as a cachedCredential object.
func readCache(filename string) (cache cachedCredential, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("unable to open file %s: %v", filename, err)
		return
	}

	err = json.Unmarshal(data, &cache)
	if err != nil {
		err = fmt.Errorf("unable to parse file %s: %v", filename, err)
	}
	return
}

// writeCache writes a cachedCredential to the specified file as json.
func writeCache(filename string, cache cachedCredential) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	// write privately owned by the user
	return ioutil.WriteFile(filename, data, 0600)
}
