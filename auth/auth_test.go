package auth

import (
	"bytes"
	"crypto/sha256"
	"os"
	"runtime"
	"testing"

	cmdutil "github.com/GGP1/kure/commands"
	"github.com/GGP1/kure/db/auth"

	"github.com/awnumar/memguard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

func TestLogin(t *testing.T) {
	db := cmdutil.SetContext(t, "../db/testdata/database")

	// This mock is used to execute Login as PreRunE
	mock := func(db *bolt.DB) *cobra.Command {
		return &cobra.Command{
			Use:     "mock",
			PreRunE: Login(db),
		}
	}

	cmd := mock(db)
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Errorf("Login() failed: %v", err)
	}
}

func TestAskArgon2Params(t *testing.T) {
	cases := []struct {
		desc            string
		input           string
		expectedIters   uint32
		expectedMem     uint32
		expectedThreads uint32
	}{
		{
			desc:            "Custom values",
			input:           "3\n2500000\n6\n",
			expectedIters:   3,
			expectedMem:     2500000,
			expectedThreads: 6,
		},
		{
			desc:            "Default values",
			input:           "\n\n\n",
			expectedIters:   1,
			expectedMem:     1048576,
			expectedThreads: uint32(runtime.NumCPU()),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			buf := bytes.NewBufferString(tc.input)

			iterations, memory, threads, err := askArgon2Params(buf)
			if err != nil {
				t.Fatalf("Failed taking argon2 parameters: %v", err)
			}

			if iterations != tc.expectedIters {
				t.Errorf("Expected %d, got %d", tc.expectedIters, iterations)
			}

			if memory != tc.expectedMem {
				t.Errorf("Expected %d, got %d", tc.expectedMem, memory)
			}

			if threads != tc.expectedThreads {
				t.Errorf("Expected %d, got %d", tc.expectedThreads, threads)
			}
		})
	}
}

func TestArgon2ParamsErrors(t *testing.T) {
	cases := []struct {
		desc  string
		input string
	}{
		{
			desc:  "iterations",
			input: "A\n",
		},
		{
			desc:  "memory",
			input: "4\nA\n",
		},
		{
			desc:  "threads",
			input: "4\n500000\nA\n",
		},
	}

	for _, tc := range cases {
		t.Run("Invalid"+tc.desc, func(t *testing.T) {
			buf := bytes.NewBufferString(tc.input)

			if _, _, _, err := askArgon2Params(buf); err == nil {
				t.Error("Expected an error and got nil")
			}
		})
	}
}

func TestAskKeyfile(t *testing.T) {
	viper.Reset()

	cases := []struct {
		desc            string
		expected        bool
		input           string
		expectedCfgPath string
	}{
		{
			desc:            "Do not use key file",
			expected:        false,
			input:           "n\n",
			expectedCfgPath: "./testdata/test-32.key",
		},
		{
			desc:            "Use key file with custom path",
			expected:        true,
			input:           "y\nn\n",
			expectedCfgPath: "",
		},
		{
			desc:            "Use key file with the config path",
			expected:        true,
			input:           "y\ny\n",
			expectedCfgPath: "./testdata/test-32.key",
		},
	}

	viper.SetConfigFile("testdata/mock_config.yaml")
	viper.SetConfigType("yaml")

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			// Set the key file path to the configuration
			viper.Set(keyfilePath, "./testdata/test-32.key")
			buf := bytes.NewBufferString(tc.input)

			got, err := askKeyfile(buf)
			if err != nil {
				t.Fatalf("Failed requesting key file: %v", err)
			}

			if got != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, got)
			}

			cfgPath := viper.Get(keyfilePath)
			if cfgPath != tc.expectedCfgPath {
				t.Errorf("Expected %q, got %q", tc.expectedCfgPath, cfgPath)
			}
		})
	}

	t.Run("Error", func(t *testing.T) {
		viper.Reset() // Unset config file and type
		viper.Set(keyfilePath, "./testdata/test-default.key")

		buf := bytes.NewBufferString("y\nn\n")
		if _, err := askKeyfile(buf); err == nil {
			t.Error("Expected 'config type could not be determined' error and got nil")
		}
	})
}

func TestCombineKeys(t *testing.T) {
	cases := []struct {
		desc string
		path string
		hash bool
	}{
		{
			desc: "32 bytes file",
			path: "./testdata/test-32.key",
			hash: false,
		},
		{
			desc: "Other file",
			path: "./testdata/test-default.key",
			hash: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			viper.Set(keyfilePath, tc.path)

			enclave, err := combineKeys(nil, memguard.NewEnclave([]byte("test")))
			if err != nil {
				t.Fatalf("Failed combining keys: %v", err)
			}

			pwdBuf, err := enclave.Open()
			if err != nil {
				t.Errorf("Failed opening enclave: %v", err)
			}
			defer pwdBuf.Destroy()

			key, err := os.ReadFile(tc.path)
			if err != nil {
				t.Errorf("Failed reading key file: %v", err)
			}

			if tc.hash {
				h := sha256.New()
				h.Write(key)
				key = h.Sum(nil)
			}

			key = append(key, []byte("test")...)

			if !bytes.Equal(pwdBuf.Bytes(), key) {
				t.Errorf("Expected %q, got %q", string(key), pwdBuf.String())
			}
		})
	}
}

func TestCombineKeysRequestPath(t *testing.T) {
	viper.Reset()
	path := "./testdata/test-32.key"
	buf := bytes.NewBufferString(path)

	enclave, err := combineKeys(buf, memguard.NewEnclave([]byte("test")))
	if err != nil {
		t.Error(err)
	}

	pwdBuf, err := enclave.Open()
	if err != nil {
		t.Errorf("Failed opening enclave: %v", err)
	}
	defer pwdBuf.Destroy()

	key, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed reading key file: %v", err)
	}

	key = append(key, []byte("test")...)

	if !bytes.Equal(pwdBuf.Bytes(), key) {
		t.Errorf("Expected %q, got %q", string(key), pwdBuf.String())
	}
}

func TestCombineKeysErrors(t *testing.T) {
	viper.Set("keyfile.path", "non-existent")

	if _, err := combineKeys(nil, memguard.NewEnclave([]byte("test"))); err == nil {
		t.Error("Expected an error and got nil")
	}

	t.Run("Invalid path", func(t *testing.T) {
		viper.Reset()
		if _, err := combineKeys(bytes.NewBufferString(""), nil); err == nil {
			t.Errorf("Expected an error and got nil")
		}
	})
}

func TestSetAuthToConfig(t *testing.T) {
	defer viper.Reset()

	expPassword := memguard.NewEnclave([]byte("test"))
	var (
		expMem  uint32 = 150000
		expIter uint32 = 110
		expTh   uint32 = 4
	)

	authParams := auth.Parameters{
		Memory:     expMem,
		Iterations: expIter,
		Threads:    expTh,
	}

	setAuthToConfig(expPassword, authParams)

	// reflect.DeepEqual does not work
	got := viper.Get("auth").(map[string]interface{})
	gotPassword := got["password"]
	gotMem := got["memory"].(uint32)
	gotIter := got["iterations"].(uint32)
	gotTh := got["threads"].(uint32)

	if gotPassword != expPassword {
		t.Errorf("Expected %#v, got %#v", expPassword, gotPassword)
	}
	if gotMem != expMem {
		t.Errorf("Expected %d, got %d", expMem, gotMem)
	}
	if gotIter != expIter {
		t.Errorf("Expected %d, got %d", expIter, gotIter)
	}
	if gotTh != expTh {
		t.Errorf("Expected %d, got %d", expTh, gotTh)
	}
}
