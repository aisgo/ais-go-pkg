package http

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/gofiber/fiber/v3"
)

/* ========================================================================
 * HTTP Listener 创建辅助函数
 * ========================================================================
 * 职责: 预先创建 net.Listener，确保端口绑定成功
 * ======================================================================== */

// createListener 根据 ListenConfig 创建 net.Listener
// 这样可以在启动 Serve 之前确保端口绑定成功
func createListener(addr string, config fiber.ListenConfig) (net.Listener, error) {
	// 确定网络类型
	network := config.ListenerNetwork
	if network == "" {
		network = "tcp4"
	}

	// 创建基础 listener
	var ln net.Listener
	var err error

	// 如果启用了 TLS
	if config.CertFile != "" && config.CertKeyFile != "" {
		// 加载 TLS 证书
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.CertKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
		}

		// 创建 TLS 配置
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		// 如果指定了最低 TLS 版本
		if config.TLSMinVersion > 0 {
			tlsConfig.MinVersion = config.TLSMinVersion
		}

		// 如果有客户端证书（mTLS）
		if config.CertClientFile != "" {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		// 创建 TLS listener
		ln, err = tls.Listen(network, addr, tlsConfig)
	} else {
		// 创建普通 TCP listener
		ln, err = net.Listen(network, addr)
	}

	if err != nil {
		return nil, err
	}

	return ln, nil
}
