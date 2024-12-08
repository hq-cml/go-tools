/*
 * 基于椭圆曲线的秘钥交换
 */
package ecdh

import (
	"bytes"
	"crypto"
	"crypto/elliptic"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/wsddn/go-ecdh"
	"hash"
	"math"
)

// 基于椭圆曲线 elliptic.P256生成私钥、公钥
func GenECDSAKey_secp256r1() (crypto.PrivateKey, []byte, error) {
	curve := ecdh.NewEllipticECDH(elliptic.P256())

	privateKey, publicKey, err := curve.GenerateKey(crand.Reader)
	if err != nil {
		return "", nil, err
	}
	pubKeyBytes := curve.Marshal(publicKey)

	// 这里pubKey之所以要转换为[]byte，主要是为了网络交互，因为pub是用来交互的，需要走网络
	return privateKey, pubKeyBytes, nil
}

// 模拟Echd椭圆取消交换秘钥的过程
func SimulateEcdh() {
	//---------------------- 服务端Hello -------------------------
	// 服务端，生成自己的椭圆密钥对
	privKeyS, pubKeyBytesS, err := GenECDSAKey_secp256r1()
	if err != nil {
		panic(err)
		return
	}

	// 公司私库，此处将开源库私自改成了EllipticPrivateKey（原版是非导出ellipticPrivateKey）
	// 为什么要这么改，因为需要在这一步能够jsonencode，然后存入redis
	//（因为交换逻辑被拆分成了几次接口交互，所以私钥要暂存redis）
	// privS := privKeyS.(*ecdh.EllipticPrivateKey)
	// TODO：将privS存入缓存
	// TODO: 将pubKeyS通过接口返回给客户端

	// ---------------------- 客户端 -------------------------
	// 客户端，生成自己的椭圆密钥对
	privKeyD, pubKeyBytesD, err := GenECDSAKey_secp256r1()
	if err != nil {
		panic(err)
		return
	}
	// 理由同上
	//privD := privKeyD.(*ecdh.EllipticPrivateKey)

	// 客户端生成shearKey
	ecdhD := ecdh.NewEllipticECDH(elliptic.P256())
	publicKeyS, _ := ecdhD.Unmarshal(pubKeyBytesS) // 从参数里面读取出pubS，并解码成为真正的pubS
	shareKeyD, err := ecdhD.GenerateSharedSecret(privKeyD, publicKeyS)
	if err != nil {
		panic(err)
	}
	// TODO 将pubKeyD通过接口请求，传递给服务端

	//---------------------- 服务端Ecdh -------------------------
	// 服务端生成shearKey
	// TODO: 把privS从换成读出来
	ecdhS := ecdh.NewEllipticECDH(elliptic.P256())
	publicKeyD, _ := ecdhS.Unmarshal(pubKeyBytesD) // 从参数里面读取出pubD，并解码成为真正的pubD
	shareKeyS, err := ecdhS.GenerateSharedSecret(privKeyS, publicKeyD)
	if err != nil {
		panic(err)
	}

	// 完成了交换，生成的ShareKey是一样的
	fmt.Println("X-------", len(shareKeyD), hex.EncodeToString(shareKeyD))
	fmt.Println("X-------", len(shareKeyS), hex.EncodeToString(shareKeyS))

	// 原理上来说，共享Key应该是完全一致的
	if hex.EncodeToString(shareKeyD) != hex.EncodeToString(shareKeyS) {
		panic("not same")
	}

	// 通常，还需要在结合HKDF进行扩展长度扩展，比如这里扩展成为65字节
	prk, okm := HKDF(sha256.New, []byte(SALT), shareKeyD, []byte(INFO), 65)
	_ = prk // 第一步抽取的结果
	if len(okm) != 65 {
		// 长度不符合预期
		panic("len != 64")
	}
	fmt.Printf("okm len:%v, detail:%v\n", len(okm), okm)
}

const (
	SALT = "Hello"
	INFO = "world"
)

// 参考：https://blog.csdn.net/inthat/article/details/130630997
// HKDF computes a PKM and OKM (where OKM is `l` bytes long) from the provided
// parameters. If `salt` is nil, a string of `0x00` bytes of length `h.Size()`
// are used as the salt. If do not want a salt at all, just use `[]byte{}` as
// the `salt` parameter.
// info:可选的上下文与应用相关信息,可为空。(用于区分不同的密钥)
func HKDF(h func() hash.Hash, salt, ikm, info []byte, l int) (prk, okm []byte) {
	if salt == nil {
		salt = bytes.Repeat([]byte{0x00}, h().Size())
	}
	f := hmac.New(h, salt)
	f.Write(ikm)
	prk = f.Sum(nil)
	hl := len(prk)
	okm = make([]byte, l, l)
	f = hmac.New(h, prk)
	for i := uint8(1); i <= uint8(math.Ceil(float64(l)/float64(hl))); i++ {
		s := int(i-2) * hl
		e := int(i-1) * hl
		if i != 1 {
			f.Write(okm[s:e])
		}
		f.Write(info)
		f.Write([]byte{i})
		copy(okm[e:], f.Sum(nil))
		f.Reset()
	}
	return
}
