package data

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"git.tcp.direct/tcp.direct/database"
)

const (
	PlaceHolderGroup  = "$g"
	PlaceHolderBridge = "$b"
	PlaceHolderLight  = "$l"
)

type TargetType uint16

const (
	TargetTypeBridge TargetType = iota
	TargetTypeGroup
	TargetTypeLight
)

func targetTypeToString(target TargetType) string {
	switch target {
	case TargetTypeGroup:
		return "group"
	case TargetTypeBridge:
		return "bridge"
	case TargetTypeLight:
		return "light"
	}
	return "unknown"
}

type Targets map[int]string

type Sequence struct {
	Lines         []string               `json:"lines"`
	TargetsNeeded map[TargetType]Targets `json:"targets_needed,omitempty"`
}

func newSequence() *Sequence {
	t := make(map[TargetType]Targets)
	t[TargetTypeGroup] = make(Targets)
	t[TargetTypeBridge] = make(Targets)
	t[TargetTypeLight] = make(Targets)
	return &Sequence{
		TargetsNeeded: t,
	}
}

func kvs() database.Store {
	return kv().With("sequences")
}

func AddSequence(sequence string, commands []string) error {
	seq := newSequence()
	prepSequence := func(placeholder string, target TargetType, i int, field string) error {
		targetID, numErr := strconv.Atoi(field[len(placeholder):])
		if numErr != nil {
			return fmt.Errorf("line %d: variable %s invalid: %w", i, field, numErr)
		}
		seq.TargetsNeeded[target][targetID] = "needed"
		return nil
	}
	for i, cmd := range commands {
		for _, field := range strings.Fields(cmd) {
			switch {
			case strings.HasPrefix(field, PlaceHolderGroup):
				if err := prepSequence(PlaceHolderGroup, TargetTypeGroup, i, field); err != nil {
					return err
				}
			case strings.HasPrefix(field, PlaceHolderBridge):
				if err := prepSequence(PlaceHolderBridge, TargetTypeBridge, i, field); err != nil {
					return err
				}
			case strings.HasPrefix(field, PlaceHolderLight):
				if err := prepSequence(PlaceHolderLight, TargetTypeLight, i, field); err != nil {
					return err
				}
			}
		}
		seq.Lines = append(seq.Lines, cmd)
	}
	seqjson, err := json.Marshal(seq)
	if err != nil {
		return err
	}
	return kvs().Put([]byte(strings.ToLower(strings.TrimSpace(sequence))), seqjson)
}

func getSequence(sequence string) (*Sequence, error) {
	seqjson, err := kvs().Get([]byte(strings.ToLower(strings.TrimSpace(sequence))))
	if err != nil {
		return nil, err
	}
	seq := newSequence()
	err = json.Unmarshal(seqjson, seq)
	if err != nil {
		return nil, err
	}
	return seq, nil
}

func ParseRunSequence(input string) (sequence string, targets map[TargetType]map[int]string, err error) {
	targets = make(map[TargetType]map[int]string)
	processArgument := func(ttype TargetType, arg, sep string) error {
		if _, ok := targets[ttype]; !ok {
			targets[ttype] = make(map[int]string)
		}
		if len(arg) == 0 {
			return fmt.Errorf("invalid %s argument: empty", targetTypeToString(ttype))
		}
		val := strings.Split(arg[1:], sep)
		targetID, numErr := strconv.Atoi(val[0])
		if numErr != nil {
			return fmt.Errorf(
				"targetID %s invalid: %w", val[0], numErr)
		}
		targets[ttype][targetID] = val[1]
		return nil
	}

	for i, arg := range strings.Fields(input) {
		if i == 0 {
			sequence = arg
			continue
		}
		arg = strings.TrimPrefix(arg, "$")
		lenColons := strings.Count(arg, ":")
		lenEquals := strings.Count(arg, "=")
		var sep = ":"
		switch {
		case lenColons == 0 && lenEquals == 0:
			return "", nil, fmt.Errorf(
				"argument %d: invalid variable assignment, missing ':' or '=': %s", i, arg)
		case lenColons > 0 && lenEquals > 0:
			return "", nil, fmt.Errorf(
				"argument %d: invalid variable assignment, cannot have both ':' and '=': %s", i, arg)
		case lenColons > 1 || lenEquals > 1:
			return "", nil, fmt.Errorf(
				"argument %d: invalid variable assignment, cannot have more than one ':' or '=': %s", i, arg)
		case lenColons == 0 || lenEquals == 1:
			sep = "="
		}
		switch {
		case strings.HasPrefix(arg, strings.TrimPrefix(PlaceHolderGroup, "$")):
			if err := processArgument(TargetTypeGroup, arg, sep); err != nil {
				return "", nil, err
			}
		case strings.HasPrefix(arg, strings.TrimPrefix(PlaceHolderBridge, "$")):
			if err := processArgument(TargetTypeBridge, arg, sep); err != nil {
				return "", nil, err
			}
		case strings.HasPrefix(arg, strings.TrimPrefix(PlaceHolderLight, "$")):
			if err := processArgument(TargetTypeLight, arg, sep); err != nil {
				return "", nil, err
			}
		default:
			return "", nil, fmt.Errorf("argument %d: invalid variable assignment: %s", i, arg)
		}
	}
	return
}

func RunSequence(sequence string, targets map[TargetType]map[int]string) ([]string, error) {
	fetchedSequence, err := getSequence(sequence)
	if err != nil {
		return nil, err
	}
	for target, ids := range fetchedSequence.TargetsNeeded {
		for id, val := range ids {
			if val != "needed" {
				continue
			}
			if _, ok := targets[target][id]; !ok {
				return nil, fmt.Errorf("sequence %s requires target %s %d",
					sequence, targetTypeToString(target), id)
			}
		}
	}
	for li, l := range fetchedSequence.Lines {
		for _, field := range strings.Fields(l) {
			var newfield string
			switch {
			case strings.HasPrefix(field, PlaceHolderGroup):
				targetID, _ := strconv.Atoi(field[len(PlaceHolderGroup):])
				newfield = targets[TargetTypeGroup][targetID]
			case strings.HasPrefix(field, PlaceHolderBridge):
				targetID, _ := strconv.Atoi(field[len(PlaceHolderBridge):])
				newfield = targets[TargetTypeBridge][targetID]
			case strings.HasPrefix(field, PlaceHolderLight):
				targetID, _ := strconv.Atoi(field[len(PlaceHolderLight):])
				newfield = targets[TargetTypeLight][targetID]
			default:
				continue
			}
			if newfield != "" {
				fetchedSequence.Lines[li] = strings.Replace(fetchedSequence.Lines[li], field, newfield, -1)
			}
		}
	}
	return fetchedSequence.Lines, nil
}
