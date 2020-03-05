package volume

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

const itemsFixture = `
#TYPE Selected.System.IO.DirectoryInfo
"FullName","Length","CreationTimeUtc","LastAccessTimeUtc","LastWriteTimeUtc","Attributes"
"C:\Program Files",,"2018-09-15T07:19:00Z","2020-01-07T21:11:01Z","2020-01-07T21:10:43Z","ReadOnly, Directory"
"C:\Windows",,"2018-09-15T06:09:26Z","2020-01-07T20:43:01Z","2020-01-07T20:43:01Z","Directory"
"C:\Program Files\Windows Mail",,"2018-09-15T07:19:00Z","2018-09-15T07:19:03Z","2018-09-15T07:19:03Z","Directory"
"C:\Windows\drivers",,"2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","Directory"
"C:\Windows\Fonts",,"2018-09-15T07:19:01Z","2019-09-07T00:21:10Z","2019-09-07T00:21:10Z","ReadOnly, System, Directory"
"C:\Windows\Prefetch",,"2019-10-13T08:15:00Z","2019-10-13T08:15:00Z","2019-10-13T08:15:00Z","Directory, NotContentIndexed"
"C:\Windows\bfsvc.exe","78848","2018-09-15T07:12:58Z","2018-09-15T07:12:58Z","2018-09-15T07:12:58Z","Archive"
"C:\Windows\bootstat.dat","67584","2019-10-13T01:16:07Z","2020-01-07T21:05:03Z","2020-01-07T21:05:03Z","System, Archive"
`

func TestStatCmdPowershell(t *testing.T) {
	const piping = " | Select-Object FullName,Length,CreationTimeUtc," +
		`LastAccessTimeUtc,LastWriteTimeUtc,Attributes | ForEach-Object {
$utc=[Xml.XmlDateTimeSerializationMode]::Utc
$_.CreationTimeUtc = [Xml.XmlConvert]::ToString($_.CreationTimeUtc,$utc)
$_.LastAccessTimeUtc = [Xml.XmlConvert]::ToString($_.LastAccessTimeUtc,$utc)
$_.LastWriteTimeUtc = [Xml.XmlConvert]::ToString($_.LastWriteTimeUtc,$utc)
$_ } | ConvertTo-Csv`

	cmd := StatCmdPowershell("", 1)
	assert.Equal(t, []string{"Get-ChildItem '/' -Recurse -Depth 0" + piping}, cmd)

	cmd = StatCmdPowershell("/", 1)
	assert.Equal(t, []string{"Get-ChildItem '/' -Recurse -Depth 0" + piping}, cmd)

	cmd = StatCmdPowershell("/Program Files/PowerShell", 5)
	assert.Equal(t, []string{"Get-ChildItem '/Program Files/PowerShell' -Recurse -Depth 4" + piping}, cmd)
}

func TestParseStatPowershell(t *testing.T) {
	dmap, err := ParseStatPowershell(strings.NewReader(itemsFixture), mountpoint, mountpoint, mountDepth)
	assert.NoError(t, err)
	assert.NotNil(t, dmap)
	assert.Len(t, dmap, 7)
	for _, dir := range []string{
		RootPath, "/Program Files", "/Windows", "/Program Files/Windows Mail",
		"/Windows/drivers", "/Windows/Fonts", "/Windows/Prefetch",
	} {
		assert.Contains(t, dmap, dir)
	}

	for _, node := range []string{"Program Files", "Windows"} {
		assert.Contains(t, dmap[RootPath], node)
	}

	expectedAttr := plugin.EntryAttributes{}
	expectedAttr.
		SetCrtime(time.Date(2018, time.September, 15, 7, 19, 0, 0, time.UTC)).
		SetAtime(time.Date(2018, time.September, 15, 7, 19, 3, 0, time.UTC)).
		SetMtime(time.Date(2018, time.September, 15, 7, 19, 3, 0, time.UTC)).
		SetCtime(time.Date(2018, time.September, 15, 7, 19, 3, 0, time.UTC)).
		SetMode(0600 | os.ModeDir).
		SetSize(0)
	assert.Equal(t, expectedAttr, dmap["/Program Files"]["Windows Mail"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetCrtime(time.Date(2018, time.September, 15, 7, 19, 1, 0, time.UTC)).
		SetAtime(time.Date(2019, time.September, 7, 0, 21, 10, 0, time.UTC)).
		SetMtime(time.Date(2019, time.September, 7, 0, 21, 10, 0, time.UTC)).
		SetCtime(time.Date(2019, time.September, 7, 0, 21, 10, 0, time.UTC)).
		SetMode(0400 | os.ModeDir).
		SetSize(0)
	assert.Equal(t, expectedAttr, dmap["/Windows"]["Fonts"])

	expectedAttr = plugin.EntryAttributes{}
	expectedAttr.
		SetCrtime(time.Date(2018, time.September, 15, 7, 12, 58, 0, time.UTC)).
		SetAtime(time.Date(2018, time.September, 15, 7, 12, 58, 0, time.UTC)).
		SetMtime(time.Date(2018, time.September, 15, 7, 12, 58, 0, time.UTC)).
		SetCtime(time.Date(2018, time.September, 15, 7, 12, 58, 0, time.UTC)).
		SetMode(0600).
		SetSize(78848)
	assert.Equal(t, expectedAttr, dmap["/Windows"]["bfsvc.exe"])
}

func TestParseStatPowershell_Empty(t *testing.T) {
	dmap, err := ParseStatPowershell(strings.NewReader(""), mountpoint, mountpoint, mountDepth)
	assert.NoError(t, err)
	assert.NotNil(t, dmap)
	assert.Len(t, dmap, 1)
	assert.Empty(t, dmap[RootPath])
}

func TestNormalErrorPowerShell(t *testing.T) {
	assert.False(t, NormalErrorPowerShell(""))
	assert.False(t, NormalErrorPowerShell("anything"))
}
