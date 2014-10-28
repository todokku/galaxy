package config

import (
	"regexp"
	"strings"
)

type Value struct {
	value interface{}
	ttl   int
}

type MemoryBackend struct {
	maps map[string]map[string]string

	MembersFunc      func(key string) ([]string, error)
	KeysFunc         func(key string) ([]string, error)
	AddMemberFunc    func(key, value string) (int, error)
	RemoveMemberFunc func(key, value string) (int, error)
	NotifyFunc       func(key, value string) (int, error)
	SetMultiFunc     func(key string, values map[string]string) (string, error)
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		maps: make(map[string]map[string]string),
	}
}

func (r *MemoryBackend) Connect() {
}

func (r *MemoryBackend) Reconnect() {
}

func (r *MemoryBackend) Keys(key string) ([]string, error) {
	if r.KeysFunc != nil {
		return r.KeysFunc(key)
	}

	keys := []string{}
	rp := strings.NewReplacer("*", `.*`)
	p := rp.Replace(key)

	re := regexp.MustCompile(p)
	for k := range r.maps {
		if re.MatchString(k) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (r *MemoryBackend) Expire(key string, ttl uint64) (int, error) {
	return 0, nil
}

func (r *MemoryBackend) Ttl(key string) (int, error) {
	return 0, nil
}

func (r *MemoryBackend) Delete(key string) (int, error) {
	if _, ok := r.maps[key]; ok {
		delete(r.maps, key)
		return 1, nil
	}
	return 0, nil
}

func (r *MemoryBackend) AddMember(key, value string) (int, error) {
	if r.AddMemberFunc != nil {
		return r.AddMemberFunc(key, value)
	}

	set := r.maps[key]
	if set == nil {
		set = make(map[string]string)
		r.maps[key] = set
	}
	set[value] = "1"
	return 1, nil
}

func (r *MemoryBackend) RemoveMember(key, value string) (int, error) {
	if r.RemoveMemberFunc != nil {
		return r.RemoveMemberFunc(key, value)
	}

	set := r.maps[key]
	if set == nil {
		return 0, nil
	}

	if _, ok := set[value]; ok {
		delete(set, value)
		return 1, nil
	}
	return 0, nil

}

func (r *MemoryBackend) Members(key string) ([]string, error) {
	if r.MembersFunc != nil {
		return r.MembersFunc(key)
	}

	values := []string{}
	set := r.maps[key]
	for v := range set {
		values = append(values, v)
	}
	return values, nil
}

func (r *MemoryBackend) Notify(key, value string) (int, error) {
	if r.NotifyFunc != nil {
		return r.NotifyFunc(key, value)
	}
	return 0, nil
}

func (r *MemoryBackend) Subscribe(key string) chan string {
	return make(chan string)
}

func (r *MemoryBackend) Set(key, field string, value string) (string, error) {
	return "OK", nil
}

func (r *MemoryBackend) Get(key, field string) (string, error) {
	return "", nil
}

func (r *MemoryBackend) GetAll(key string) (map[string]string, error) {
	return r.maps[key], nil
}

func (r *MemoryBackend) SetMulti(key string, values map[string]string) (string, error) {
	if r.SetMultiFunc != nil {
		return r.SetMultiFunc(key, values)
	}

	r.maps[key] = values
	return "OK", nil
}

func (r *MemoryBackend) DeleteMulti(key string, fields ...string) (int, error) {
	m := r.maps[key]
	for _, field := range fields {
		delete(m, field)
	}
	return len(fields), nil
}