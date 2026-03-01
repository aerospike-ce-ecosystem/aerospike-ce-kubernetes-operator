package initcontainer

import (
	"strings"
	"testing"
)

func TestGetInitScript(t *testing.T) {
	script := GetInitScript()

	if script == "" {
		t.Fatal("GetInitScript() returned empty string")
	}

	// Verify the script starts with a shebang
	if !strings.HasPrefix(script, "#!/bin/bash") {
		t.Error("init script does not start with #!/bin/bash shebang")
	}

	// Verify key operations are present
	for _, expected := range []string{
		"CONFIG_SRC",
		"CONFIG_DST",
		"aerospike.conf",
		"MY_POD_IP",
		"WIPE_VOLUMES",
		"INIT_VOLUMES",
	} {
		if !strings.Contains(script, expected) {
			t.Errorf("init script missing expected content: %q", expected)
		}
	}
}

func TestGetConfigMapData(t *testing.T) {
	conf := "service { cluster-name test }"
	data := GetConfigMapData(conf)

	if len(data) != 2 {
		t.Fatalf("GetConfigMapData() returned %d entries, want 2", len(data))
	}

	if got := data["aerospike.conf"]; got != conf {
		t.Errorf("aerospike.conf = %q, want %q", got, conf)
	}

	initScript := data["aerospike-init.sh"]
	if initScript == "" {
		t.Fatal("aerospike-init.sh is empty")
	}

	// The init script in ConfigMap should match GetInitScript()
	if initScript != GetInitScript() {
		t.Error("aerospike-init.sh in ConfigMap data does not match GetInitScript()")
	}
}

func TestGetConfigMapData_EmptyConf(t *testing.T) {
	data := GetConfigMapData("")

	if got := data["aerospike.conf"]; got != "" {
		t.Errorf("aerospike.conf = %q, want empty string", got)
	}

	if data["aerospike-init.sh"] == "" {
		t.Error("aerospike-init.sh should still contain the init script even with empty conf")
	}
}
