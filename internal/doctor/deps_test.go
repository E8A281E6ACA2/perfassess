package doctor

import "testing"

func TestCheckDependencyReportsMissingTool(t *testing.T) {
	status := checkDependency("definitely-not-installed-perfassess-tool", "测试用途", "安装提示", true)

	if status.Found {
		t.Fatal("expected missing dependency")
	}
	if status.Name != "definitely-not-installed-perfassess-tool" {
		t.Fatalf("unexpected dependency name: %s", status.Name)
	}
	if status.Hint != "安装提示" {
		t.Fatalf("unexpected hint: %s", status.Hint)
	}
	if !status.Required {
		t.Fatal("expected required flag to be preserved")
	}
}

func TestCheckDependencyReportsExistingTool(t *testing.T) {
	status := checkDependency("go", "Go 工具链", "安装 Go", false)

	if !status.Found {
		t.Skip("go binary is not available in PATH")
	}
	if status.Path == "" {
		t.Fatal("expected existing dependency path")
	}
}
