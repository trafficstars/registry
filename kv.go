package registry

import (
	"github.com/hashicorp/consul/api"
)

type kv struct {
	client *api.KV
}

func (kv *kv) Get(key string) (string, error) {
	v, _, err := kv.client.Get(REGISTRY_PREFIX+"/"+key, nil)
	if err != nil {
		return "", err
	}
	if v != nil {
		return string(v.Value), nil
	}
	return "", nil
}

func (kv *kv) Set(key, value string) error {
	if _, err := kv.client.Put(&api.KVPair{Key: REGISTRY_PREFIX + "/" + key, Value: []byte(value)}, nil); err != nil {
		return err
	}
	return nil
}

func (kv *kv) List(prefix string) (map[string]string, error) {
	items, _, err := kv.client.List(prefix, nil)
	if err != nil {
		return nil, err
	}
	list := make(map[string]string, len(items))
	for _, k := range items {
		list[k.Key] = string(k.Value)
	}
	return list, nil
}

func (kv *kv) Delete(key string) error {
	if _, err := kv.client.Delete(REGISTRY_PREFIX+"/"+key, nil); err != nil {
		return err
	}
	return nil
}
