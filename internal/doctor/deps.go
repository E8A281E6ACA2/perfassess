package doctor

import (
	"fmt"
	"os/exec"
	"runtime"
)

type DependencyStatus struct {
	Name     string
	Purpose  string
	Found    bool
	Path     string
	Hint     string
	Required bool
}

func CheckDependencies() []DependencyStatus {
	deps := []DependencyStatus{
		checkDependency("iperf3", "iperf3 网络吞吐测试", iperf3InstallHint(), false),
		checkDependency("fio", "fio 磁盘基准测试", fioInstallHint(), false),
	}

	routeBin := "traceroute"
	routeHint := routeTraceInstallHint(routeBin)
	if runtime.GOOS == "windows" {
		routeBin = "tracert"
		routeHint = routeTraceInstallHint(routeBin)
	}
	deps = append(deps, checkDependency(routeBin, "路由追踪测试", routeHint, false))

	return deps
}

func checkDependency(name string, purpose string, hint string, required bool) DependencyStatus {
	path, err := exec.LookPath(name)
	if err != nil {
		return DependencyStatus{
			Name:     name,
			Purpose:  purpose,
			Found:    false,
			Hint:     hint,
			Required: required,
		}
	}

	return DependencyStatus{
		Name:     name,
		Purpose:  purpose,
		Found:    true,
		Path:     path,
		Required: required,
	}
}

func iperf3InstallHint() string {
	switch runtime.GOOS {
	case "linux":
		return "Ubuntu/Debian: sudo apt install iperf3；RHEL/CentOS: sudo yum install iperf3"
	case "darwin":
		return "macOS: brew install iperf3"
	case "windows":
		return "Windows: 请通过 winget/choco 或 iperf.fr 安装 iperf3，并确认 PATH 可访问"
	default:
		return "请安装 iperf3，并确认 PATH 可访问"
	}
}

func fioInstallHint() string {
	switch runtime.GOOS {
	case "linux":
		return "Ubuntu/Debian: sudo apt install fio；RHEL/CentOS: sudo yum install fio"
	case "darwin":
		return "macOS: brew install fio"
	case "windows":
		return "Windows: 请通过 winget/choco 或 fio 官方发行包安装 fio，并确认 PATH 可访问"
	default:
		return "请安装 fio，并确认 PATH 可访问"
	}
}

func routeTraceInstallHint(bin string) string {
	switch runtime.GOOS {
	case "linux":
		return "Ubuntu/Debian: sudo apt install traceroute；RHEL/CentOS: sudo yum install traceroute"
	case "darwin":
		return "macOS 通常自带 traceroute；如不可用，请确认系统 PATH"
	case "windows":
		if bin == "tracert" {
			return "Windows 通常自带 tracert；如不可用，请确认系统 PATH 或管理员策略"
		}
	}
	return fmt.Sprintf("请安装 %s，并确认 PATH 可访问", bin)
}
