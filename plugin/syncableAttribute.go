package plugin

import (
	"fmt"
	"time"
)

// AtimeAttr represents the Atime attribute in plugin.EntryAttributes
func AtimeAttr() SyncableAttribute {
	return atimeField
}

// MtimeAttr represents the Mtime attribute in plugin.EntryAttributes
func MtimeAttr() SyncableAttribute {
	return mtimeField
}

// CtimeAttr represents the Ctime attribute in plugin.EntryAttributes
func CtimeAttr() SyncableAttribute {
	return ctimeField
}

// ModeAttr represents the Mode attribute in plugin.EntryAttributes
func ModeAttr() SyncableAttribute {
	return modeField
}

// SizeAttr represents the Size attribute in plugin.EntryAttributes
func SizeAttr() SyncableAttribute {
	return sizeField
}

// SyncableAttribute represents a syncable attribute in
// plugin.EntryAttributes. It abstracts away the common logic of
// syncing the attribute with its corresponding key in the entry's
// metadata whenever the latter's refreshed.
type SyncableAttribute struct {
	name   string
	setter func(*EntryBase, interface{}) error
}

func (field SyncableAttribute) sync(entry *EntryBase, meta EntryMetadata, key string) error {
	value, ok := meta[key]
	if !ok {
		return fmt.Errorf(
			"the metadata does not contain a value in its %v key",
			key,
		)
	}
	if value == nil {
		return fmt.Errorf(
			"the metadata's %v key is set to null",
			key,
		)
	}
	if err := field.setter(entry, value); err != nil {
		return fmt.Errorf(
			"failed to munge the metadata's %v key: %v",
			key,
			err,
		)
	}

	return nil
}

var atimeField = SyncableAttribute{
	name: "atime",
	setter: func(entry *EntryBase, v interface{}) error {
		t, err := mungeToTimeVal(v)
		if err != nil {
			return err
		}

		entry.attr.SetAtime(t)
		return nil
	},
}

var mtimeField = SyncableAttribute{
	name: "mtime",
	setter: func(entry *EntryBase, v interface{}) error {
		t, err := mungeToTimeVal(v)
		if err != nil {
			return err
		}

		entry.attr.SetMtime(t)
		return nil
	},
}

var ctimeField = SyncableAttribute{
	name: "ctime",
	setter: func(entry *EntryBase, v interface{}) error {
		t, err := mungeToTimeVal(v)
		if err != nil {
			return err
		}

		entry.attr.SetCtime(t)
		return nil
	},
}

var modeField = SyncableAttribute{
	name: "mode",
	setter: func(entry *EntryBase, v interface{}) error {
		m, err := ToFileMode(v)
		if err != nil {
			return err
		}

		entry.attr.SetMode(m)
		return nil
	},
}

var sizeField = SyncableAttribute{
	name: "size",
	setter: func(entry *EntryBase, v interface{}) error {
		size, err := mungeToSizeVal(v)
		if err != nil {
			return err
		}

		entry.attr.SetSize(size)
		return nil
	},
}

func mungeToTimeVal(v interface{}) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		// TODO: The layout should be specified by the plugin author.
		// We could likely have helpers that generate a munging function
		// given some parameters. For now, this is enough.
		layout := "2006-01-02T15:04:05Z"
		return time.Parse(layout, t)
	default:
		return time.Time{}, fmt.Errorf("%v is not a time.Time value", v)
	}
}

func mungeIntSize(size int64) (uint64, error) {
	if size < 0 {
		return 0, fmt.Errorf("%v is a negative size", size)
	}

	return uint64(size), nil
}

func mungeToSizeVal(v interface{}) (uint64, error) {
	switch sz := v.(type) {
	case uint64:
		return sz, nil
	case int:
		return mungeIntSize(int64(sz))
	case int32:
		return mungeIntSize(int64(sz))
	case int64:
		return mungeIntSize(sz)
	case float64:
		return mungeIntSize(int64(sz))
	default:
		return 0, fmt.Errorf("%v is not a valid size type. Valid size types are uint64, int, int64", v)
	}
}
