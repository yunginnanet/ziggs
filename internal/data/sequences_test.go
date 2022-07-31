package data

import (
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"
)

var onceT = &sync.Once{}

func Setup(t *testing.T) {
	testMode()
	setTarget()
	t.Logf("testing directory: %s", target)
	t.Cleanup(func() {
		err := os.RemoveAll(target)
		if err != nil {
			println(err.Error())
			panic(err)
		}
	})
}

func mustMarshalIndent(data any, prefix string) string {
	indented, err := json.MarshalIndent(data, prefix, "\t")
	if err != nil {
		println(err.Error())
		panic(err)
	}
	return string(indented)
}

func TestAddSequence(t *testing.T) {
	onceT.Do(func() { Setup(t) })
	type args struct {
		Seq  string
		Coms []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "AddSequence",
			args: args{Seq: "test", Coms: []string{
				"set g $g1 color #8cabff",
				"set g $g1 brightness 255",
				"set g $g3 brightness 150",
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("running %s", tt.name)
			t.Log(mustMarshalIndent(tt.args, "args:"))
			if err := AddSequence(tt.args.Seq, tt.args.Coms); (err != nil) != tt.wantErr {
				t.Errorf("AddSequence() error = %v, wantErr %v", err, tt.wantErr)
			}
			fetch, newErr := getSequence(tt.args.Seq)
			if newErr != nil && !tt.wantErr {
				t.Errorf("getSequence() error = %v, wantErr %v", newErr, tt.wantErr)
			}
			t.Log(mustMarshalIndent(fetch, "fetch:"))
		})
	}
}

func TestRunSequence(t *testing.T) { //nolint:funlen
	type args struct {
		sequence string
		targets  map[TargetType]map[int]string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "RunSequence",
			args: args{
				sequence: "test",
				targets: map[TargetType]map[int]string{
					TargetTypeGroup: {
						1: "kayos",
						2: "flapjacks",
						3: "kayos",
					},
				},
			},
			want: []string{
				"set g kayos color #8cabff",
				"set g kayos brightness 255",
				"set g kayos brightness 150",
			},
			wantErr: false,
		},
		{
			name: "RunSequenceAlt",
			args: args{
				sequence: "test",
				targets: map[TargetType]map[int]string{
					TargetTypeGroup: {
						1: "kayos",
						2: "billy",
						3: "flapjacks",
					},
				},
			},
			want: []string{
				"set g kayos color #8cabff",
				"set g kayos brightness 255",
				"set g flapjacks brightness 150",
			},
			wantErr: false,
		},
		{
			name: "RunSequenceFailType",
			args: args{
				sequence: "test",
				targets: map[TargetType]map[int]string{
					TargetTypeLight: {
						1: "kayos",
						2: "flapjacks",
						3: "kayos",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "RunSequenceFailID",
			args: args{
				sequence: "test",
				targets: map[TargetType]map[int]string{
					TargetTypeLight: {
						5: "kayos",
						7: "flapjacks",
						3: "kayos",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RunSequence(tt.args.sequence, tt.args.targets)
			t.Log(mustMarshalIndent(got, "butgot:"))
			if (err != nil) != tt.wantErr {
				t.Errorf("RunSequence() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Log(mustMarshalIndent(got, "wanted:"))
				t.Errorf("RunSequence() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRunSequence(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name         string
		args         args
		wantSequence string
		wantTargets  map[TargetType]map[int]string
		wantErr      bool
	}{
		{
			name:         "ParseRunSequence",
			args:         args{input: "test g1:kayos g2:billy g3:flapjacks"},
			wantSequence: "test",
			wantTargets: map[TargetType]map[int]string{
				TargetTypeGroup: {
					1: "kayos",
					2: "billy",
					3: "flapjacks",
				},
			},
			wantErr: false,
		},
		{
			name:         "ParseRunSequenceAlt",
			args:         args{input: "test g1:kayos l2=billy $g3:flapjacks"},
			wantSequence: "test",
			wantTargets: map[TargetType]map[int]string{
				TargetTypeGroup: {
					1: "kayos",
					3: "flapjacks",
				},
				TargetTypeLight: {
					2: "billy",
				},
			},
			wantErr: false,
		},
		{
			name:    "ParseRunSequenceFail",
			args:    args{input: "test x1:kayos y2=billy $g3:flapjacks"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSequence, gotTargets, err := ParseRunSequence(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRunSequence() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSequence != tt.wantSequence {
				t.Errorf("ParseRunSequence() gotSequence = %v, want %v", gotSequence, tt.wantSequence)
			}
			if !reflect.DeepEqual(gotTargets, tt.wantTargets) {
				t.Errorf("ParseRunSequence() gotTargets = %v, want %v", gotTargets, tt.wantTargets)
			}
		})
	}
}
