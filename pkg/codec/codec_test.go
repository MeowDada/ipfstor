package codec

import (
	"fmt"
	"testing"
)

func getTestTargets() []Instance {
	return []Instance{
		Gob{},
	}
}

func TestMarshal(t *testing.T) {

	testcases := []struct {
		description string
		run         func() error
	}{
		{
			"Marshal nil object, expect no errors",
			func() error {
				targets := getTestTargets()
				for i := range targets {
					name := targets[i].Name()
					data, err := targets[i].Marshal(nil)
					if err != nil {
						return fmt.Errorf("%s: %v", name, err)
					}
					if err != nil {
						return fmt.Errorf("%s: expect return nil, but get %v", name, data)
					}
				}
				return nil
			},
		},
	}

	for _, tc := range testcases {
		if err := tc.run(); err != nil {
			t.Errorf("%s: %v", tc.description, err)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	testcases := []struct {
		description string
		run         func() error
	}{
		{
			"Unmarshal nil object",
			func() error {
				targets := getTestTargets()
				for i := range targets {
					name := targets[i].Name()
					err := targets[i].Unmarshal(nil, nil)
					if err == nil {
						return fmt.Errorf("%s: expect error, but get no errors", name)
					}
				}
				return nil
			},
		},
	}

	for _, tc := range testcases {
		if err := tc.run(); err != nil {
			t.Errorf("%s: %v", tc.description, err)
		}
	}
}
