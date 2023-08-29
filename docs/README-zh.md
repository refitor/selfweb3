# selfweb3

### 一种为链上高度安全和数据所有权提供私有化保障的web3解决方案，通过强制动态授权，将用户资产一对一绑定到自己，而非钱包，webAuthn + TOTP + 零知识证明

## 合约

1. **Arbitrum Goerli, 421613: 0x7B6E05a55B1756f827F205BF454BF75288904ecF**

## 系统架构

![/docs/selfweb3.png](/docs/selfweb3.png)

#### 说明: 公私钥指椭圆曲线secp256k1生成的公私钥对，密钥指32位明文字符串，dhKey指通过私钥和公钥动态计算出来的共享密钥

> - 合约: 去中心化存储，一经注册写入，无法更改，支持钱包重新绑定，依赖于用户私有动态授权，而非钱包公私钥

> - wasm: 提供密钥合成、TOTP校验和邮件校验服务，支持手动模式恢复web2私钥，所有操作需要依赖web2密文已解密

> - 后端: web2服务, 提供webAuthn认证，邮件发送以及加密数据存储和提取服务, 无会话，无状态，仅处理密文数据

> - TOTP: 由web2服务提供TOTP密钥密文，以及wasm部分解密出web3公钥，计算和解密出TOTP密钥后进行动态校验

> - 邮件: 由wasm负责缓存动态验证码并完成验证, web2服务负责发送

> - 前端: web2、web3交互网关，所有web2服务以及web3 dapp入口

## 特性

### web3私有化(selfWeb3)

1. 强制将web3链上操作权绑定到用户自己，web3数据所有权真正归用户自己所有

2. 通过webAuthn、TOTP和邮箱校验等方式 以及由零知识证明链下生成链上校验为强制性提高保障

3. 由于私有动态授权的强制性绑定，钱包仅作为工具与web3合约交互，但默认也支持链下请求签名并由链上合约进行校验, 支持私有动态授权后重新绑定钱包

### 用户数据所有权私有化(web3合约)

1. 合约负责存储用于动态授权校验的密钥密文，包括恢复ID签名(web3私钥签署，web3私钥丢弃), web3密钥密文(dhKey加密), web3公钥密文(web2公钥加密)

2. 负责动态校验用户身份以及操作合法性进行确权，包括钱包签名校验和零知识证明链上校验

3. 负责提供具体的web3业务, 包括私有金库、私有NFT集以及用户其他web3数据等

### 隐私安全计算(WebAssembly)

1. wasm部分作为隐私安全计算节点，负责处理所有明文计算任务以及动态校验逻辑，同时所有操作依赖于支持自行重置的web2服务密钥，用于支持用户数据隐私安全

2. wasm文件合法性暂时由官方保证，后续支持由web2服务以及web3合约协作完成动态校验，包括链下零知识证明生成以及链上动态验证

3. webAuthn数据由web2服务采用随机密钥加密存储，随机密钥解密依赖于链上的web3密钥，该密钥提取需要通过链上动态校验

4. web2服务与wasm部分采用临时非对称ecdsa密钥对计算共享密钥的形式进行点对点加密通讯

### 零信任web2服务

1. 核心代码开源，支持服务私有部署

2. 为了保证高强度安全性，仅提供针对密文相关的的无状态服务，没有会话

3. 仅维护web2相关密文数据，无法访问web3私钥和自行设置的web2服务密钥

4. 作为零信任中心化服务，仅提供web2相关密文数据存储提取，webAuthn动态授权，以及基于用户密文的定向业务增值服务

## 核心业务流程

### web2读写模式

1. 支持向web3合约和web2服务进行数据读写，动态验证流程需要关联验证通过，即email -> TOTP -> webAuthn

2. web2服务密钥是由后端web2服务生成的随机密钥，同时支持自行输入32位字符串作为web2服务密钥，但每次加载web都需要输入

3. web3公钥由web2公钥加密后存储在web3合约中, 用于与web2私钥钥动态计算合成dhKey, 一经注册写入合约无法更改

4. web3密钥由dhKey加密后存储在web3合约中，用于用户数据加密后存储到selfWeb3的web2服务，比如webAuthn相关数据

5. web2私钥由wasm部分随机生成，用于与web3公钥动态计算合成dhKey, 由web2服务密钥加密并抄送到用户邮箱

6. 重置功能包括TOTP, web2服务密钥以及用户钱包，所有重置操作都会启用完整的关联验证流程用于加强安全防护

7. dapp业务，所有需要写入web3合约的相关操作都需要TOTP和webAuthn动态校验，敏感操作将触发邮件校验

### web3只读模式

1. web3模式是在web2服务不可用或者存在安全风险的情况下支持启用的紧急处理方案，仅仅支持TOTP校验以及web3合约的钱包签名校验，不支持所有恢复重置功能

2. 由于没有web2服务支持，需要用户自行输入web2私钥密文以及可选的web2服务密钥(如果自行指定重置过web2服务密钥)

3. 所有dapp业务仅支持只读web3合约数据

### 私有部署

```shell
git clone https://github.com/refitor/selfweb3.git

cd selfweb3

chmod +x ./build.sh

./build.sh
```

### 使用

```
./selfweb3
```