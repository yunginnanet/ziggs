package data

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"git.tcp.direct/kayos/common/squish"
	"git.tcp.direct/tcp.direct/database"
)

func kvMacros() database.Store {
	return kv().With("macros")
}

type Macro struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Sequence    []string `json:"sequence"`
}

type macroCache struct {
	cache map[string]*Macro
	*sync.RWMutex
}

var macros = &macroCache{
	cache:   make(map[string]*Macro),
	RWMutex: new(sync.RWMutex),
}

func hasMacroCached(name string) *Macro {
	macros.RLock()
	defer macros.RUnlock()
	if macroPtr, ok := macros.cache[name]; ok {
		return macroPtr
	}
	return nil
}

func unpackMacro(input []byte) (macro *Macro, err error) {
	if input, err = squish.Gunzip(input); err != nil {
		return nil, fmt.Errorf("error deflating macro: %w", err)
	}
	err = json.Unmarshal(input, &macro)
	return
}

func updateCache(name string, macro *Macro) {
	macros.Lock()
	macros.cache[name] = macro
	macros.Unlock()
}

func GetMacro(name string) (mcro *Macro, err error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if cached := hasMacroCached(name); cached != nil {
		return cached, nil
	}

	if !kvMacros().Has([]byte(name)) {

	}

	var packed []byte
	if packed, err = kvMacros().Get([]byte(name)); err != nil {
		return nil, fmt.Errorf("error fetching macro: %w", err)
	}

	if mcro, err = unpackMacro(packed); err != nil {
		return nil, fmt.Errorf("error unpacking macro: %w", err)
	}

	go updateCache(name, mcro)

	return mcro, err
}

func DeleteMacro(name string) error {
	macros.Lock()
	defer macros.Unlock()
	delete(macros.cache, name)
	return fmt.Errorf(
		"failed to delete macro: %w",
		kvMacros().Delete([]byte(strings.ToLower(strings.TrimSpace(name)))),
	)
}

// AddMacro adds a macro to the database, the description is optional.
func AddMacro(name string, description string, sequence ...string) error {
	if _, err := GetMacro(name); err == nil {
		return fmt.Errorf("a macro named %q already exists", name)
	}
	mcro := Macro{
		Name:        name,
		Description: description,
		Sequence:    sequence,
	}
	rawMacro, err := json.Marshal(mcro)
	if err != nil {
		return fmt.Errorf("failed to marshal macro: %w", err)
	}
	rawMacro = squish.Gzip(rawMacro)
	return kvMacros().Put([]byte(strings.ToLower(strings.TrimSpace(name))), rawMacro)
}
