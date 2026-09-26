package service

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	tsproxy "apeadmin-gin/internal/plugin/builtin/tailscale/proxy"

	"golang.org/x/crypto/ssh"
)

// ExecuteSSHCommand 经由 tsnet 内存隧道直连目标节点执行 SSH 维护指令
func (s *TailscaleService) ExecuteSSHCommand(target string, port int, user, password, privateKey, command string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("目标地址不能为空")
	}
	if port <= 0 {
		port = 22
	}
	if user == "" {
		user = "root"
	}
	if command == "" {
		return "", fmt.Errorf("执行命令不能为空")
	}

	var authMethods []ssh.AuthMethod
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}
	if privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	if len(authMethods) == 0 {
		// 尝试 Tailscale SSH 免密握手
		authMethods = append(authMethods, ssh.Password(""))
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", target, port)

	// 1. 获取底层连接 (优先 tsnet 内存拨号，若未配置则尝试通用 SOCKS5/直接拨号)
	var conn net.Conn
	var err error
	tsnetMgr := tsproxy.GetTsnetManager()
	conn, err = tsnetMgr.DialTimeout("tcp", address, 5*time.Second)

	if err != nil {
		cfg, _ := s.GetConfig()
		tip := ""
		if cfg == nil || (cfg.ProxyURL == "" && cfg.AuthKey == "") {
			tip = "【诊断提示】：云端容器（Koyeb）自身不在局域网内。请在 [连接与嵌入式节点配置] 填入 SOCKS5 代理地址（例如您本地或服务器开的 socks5://xxx:1055）以穿透内网。"
		} else {
			tip = "【诊断提示】：目标主机 100.74.17.86 未开启 22 端口监听，或本地防火墙未放行来自 Tailscale 网卡的入站流量。"
		}
		return "", fmt.Errorf("连接目标 SSH 端口 %s 失败: %w\n%s", address, err, tip)
	}
	defer conn.Close()

	// 2. 建立 SSH Client
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH 鉴权握手失败: %w", err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	// 3. 创建 Session 执行指令
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	if err := session.Run(command); err != nil {
		combined := strings.TrimSpace(stdoutBuf.String() + "\n" + stderrBuf.String())
		return combined, fmt.Errorf("命令执行失败 [%v]: %s", err, combined)
	}

	output := stdoutBuf.String()
	if output == "" && stderrBuf.Len() > 0 {
		output = stderrBuf.String()
	}

	return strings.TrimSpace(output), nil
}
