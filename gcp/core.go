package gcp

import (
	"bytes"
	"context"
	"encoding/gob"
	"net/http"
	"sort"
	"time"

	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	"golang.org/x/oauth2/google"
	crm "google.golang.org/api/cloudresourcemanager/v1"
)

type client struct {
	oauthClient *http.Client
	lister      *crm.ProjectsListCall
	projects    map[string]*project
	cache       *bigcache.BigCache
	updated     time.Time
	name        string
}

// Defines how quickly we should allow checks for updated content. This has to be consistent
// across files and directories or we may not detect updates quickly enough, especially for files
// that previously were empty.
const validDuration = 100 * time.Millisecond

// Create a new gcp client.
func Create(name string) (plugin.DirProtocol, error) {
	cacheconfig := bigcache.DefaultConfig(5 * time.Second)
	cacheconfig.CleanWindow = 100 * time.Millisecond
	cache, err := bigcache.NewBigCache(cacheconfig)
	if err != nil {
		return nil, err
	}

	// This API is terrible, but not supported by the better go sdk.
	cloudPlatformScopes := append([]string{crm.CloudPlatformScope}, serviceScopes...)
	oauthClient, err := google.DefaultClient(context.Background(), cloudPlatformScopes...)
	if err != nil {
		return nil, err
	}
	crmService, err := crm.New(oauthClient)
	if err != nil {
		return nil, err
	}
	lister := crm.NewProjectsService(crmService).List()

	projmap := make(map[string]*project)
	return &client{oauthClient, lister, projmap, cache, time.Now(), name}, nil
}

// Find project by name.
func (cli *client) Find(ctx context.Context, name string) (plugin.Node, error) {
	cli.refreshProjects(ctx)
	if proj, ok := cli.projects[name]; ok {
		log.Debugf("Found project %v", name)
		return plugin.NewDir(proj), nil
	}
	return nil, plugin.ENOENT
}

// List all projects as dirs.
func (cli *client) List(ctx context.Context) ([]plugin.Node, error) {
	cli.refreshProjects(ctx)
	log.Debugf("Listing %v projects in /gcp", len(cli.projects))
	entries := make([]plugin.Node, 0, len(cli.projects))
	for _, proj := range cli.projects {
		entries = append(entries, plugin.NewDir(proj))
	}
	return entries, nil
}

// Name returns the root directory of the client.
func (cli *client) Name() string {
	return cli.name
}

// Attr returns attributes of the named project.
func (cli *client) Attr(ctx context.Context) (*plugin.Attributes, error) {
	latest := cli.updated
	for _, proj := range cli.projects {
		if updated := proj.lastUpdate(); updated.After(latest) {
			latest = updated
		}
	}
	return &plugin.Attributes{Mtime: latest, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *client) Xattr(ctx context.Context) (map[string][]byte, error) {
	return nil, plugin.ENOTSUP
}

func (cli *client) cachedProjectsList(ctx context.Context) ([]string, error) {
	entry, err := cli.cache.Get("ProjectsList")
	if err == nil {
		log.Debugf("Cache hit in /gcp")
		var projects []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&projects)
		return projects, err
	}

	log.Debugf("Cache miss in /gcp")
	listResponse, err := cli.lister.Do()
	if err != nil {
		return nil, err
	}

	projects := make([]string, len(listResponse.Projects))
	for i, proj := range listResponse.Projects {
		projects[i] = proj.ProjectId
	}
	sort.Strings(projects)

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&projects); err != nil {
		return nil, err
	}
	cli.cache.Set("ProjectsList", data.Bytes())
	cli.updated = time.Now()
	return projects, nil
}

func (cli *client) refreshProjects(ctx context.Context) error {
	projectNames, err := cli.cachedProjectsList(ctx)
	if err != nil {
		return err
	}

	// Remove unnamed projects
	for name, proj := range cli.projects {
		idx := sort.SearchStrings(projectNames, name)
		if projectNames[idx] != name {
			proj.closeServices(ctx)
			delete(cli.projects, name)
		}
	}

	for _, proj := range projectNames {
		if _, ok := cli.projects[proj]; ok {
			continue
		}

		newProj, err := newProject(proj, cli.oauthClient, cli.cache)
		if err != nil {
			return err
		}

		cli.projects[proj] = newProj
	}
	return nil
}
