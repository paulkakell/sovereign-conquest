package build

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAPIRuntimeDoesNotShipOSPackages(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	for _, name := range []string{"Dockerfile", "server/Dockerfile"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, name))
			if err != nil {
				t.Fatal(err)
			}
			dockerfile := string(data)
			lastStage := strings.LastIndex(dockerfile, "\nFROM ")
			if lastStage == -1 || !strings.HasPrefix(dockerfile[lastStage:], "\nFROM scratch\n") {
				t.Fatal("API runtime must start from scratch")
			}
			final := dockerfile[lastStage:]
			for _, required := range []string{
				"COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt",
				"COPY --from=build /usr/share/zoneinfo/ /usr/share/zoneinfo/",
				"COPY --from=build /out/runtime/ /",
				"COPY --from=build --chown=10001:10001 /out/sovereign-api /app/sovereign-api",
				"USER 10001:10001",
				"CMD [\"/app/sovereign-api\", \"healthcheck\"]",
				"CMD [\"/app/sovereign-api\"]",
			} {
				if !strings.Contains(final, required) {
					t.Errorf("missing runtime contract: %s", required)
				}
			}
			for _, forbidden := range []string{"\nRUN ", "wget", "busybox", "apk ", "/bin/sh", "COPY --from=build / /"} {
				if strings.Contains(final, forbidden) {
					t.Errorf("unexpected runtime content: %s", forbidden)
				}
			}
			if !strings.Contains(dockerfile[:lastStage], "RUN update-ca-certificates") {
				t.Error("runtime CA bundle must include configured custom roots")
			}
			for _, required := range []string{"mkdir -p /out/runtime/tmp", "chmod 1777 /out/runtime/tmp"} {
				if !strings.Contains(dockerfile[:lastStage], required) {
					t.Errorf("missing temporary-directory preparation: %s", required)
				}
			}
		})
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "server/scripts/build_api.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "CGO_ENABLED=0 GOOS=linux") {
		t.Error("scratch runtime requires a statically linked API")
	}
}
