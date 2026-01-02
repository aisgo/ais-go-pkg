package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/xdg-go/scram"
)

/* ========================================================================
 * SCRAM 认证支持
 * ========================================================================
 * 职责: 提供 Kafka SCRAM 认证能力
 * ======================================================================== */

// HashGeneratorFcn SCRAM hash 生成器
type HashGeneratorFcn func() hash.Hash

// SHA256 SHA256 hash 生成器
var SHA256 HashGeneratorFcn = sha256.New

// SHA512 SHA512 hash 生成器
var SHA512 HashGeneratorFcn = sha512.New

// XDGSCRAMClient SCRAM 客户端实现
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	HashGeneratorFcn HashGeneratorFcn
}

// Begin 开始 SCRAM 认证
func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	// 根据 HashGeneratorFcn 选择算法
	if x.HashGeneratorFcn != nil {
		// 通过比较 hash 结果判断是 SHA256 还是 SHA512
		testHash := x.HashGeneratorFcn()
		if testHash.Size() == 64 { // SHA512
			x.Client, err = scram.SHA512.NewClient(userName, password, authzID)
		} else { // SHA256
			x.Client, err = scram.SHA256.NewClient(userName, password, authzID)
		}
	} else {
		x.Client, err = scram.SHA256.NewClient(userName, password, authzID)
	}
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

// Step 执行 SCRAM 认证步骤
func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return
}

// Done 判断 SCRAM 认证是否完成
func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}
