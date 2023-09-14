// 处理所有用户相关的业务逻辑, 包括Load, Register, Recover, Reset等

import Web3 from "web3";
import * as selfweb3 from './index.js';

export function Init(walletAddress, inputWeb2Key, callback) {
    initBackend('Init', walletAddress, inputWeb2Key, callback);
}

export function Registered(walletAddress, selfAddress, callback) {
    selfweb3.GetWeb3().Web3Execute("call", "Registered", walletAddress, 0, [selfAddress], function (loadResult) {
        selfweb3.ShowMsg('', 'Registered', 'web3 contract call successed: ', loadResult);
        if (callback !== undefined && callback !== null) callback(loadResult, true);//(loadResult['registered'], true);
    }, function (err) {
        selfweb3.ShowMsg('error', 'Registered', 'web3 contract call failed: ', err);
    });
}

export function Load(walletAddress, selfAddress, web2Address, callback) {
    initWeb3('Load', walletAddress, selfAddress, web2Address, callback);
}

function initWeb3(flow, walletAddress, selfAddress, web2Address, callback) {
    selfweb3.ShowWaitting(true);
    selfweb3.SetProps("selfAddress", selfAddress);
    selfweb3.SetProps("web2Address", web2Address);
    let message = 'SelfWeb3 Init: ' + (new Date()).getTime();
    selfweb3.GetWeb3().Sign(flow, walletAddress, message, function(sig) {
        var loadParams = [];
        loadParams.push(selfAddress);
        loadParams.push(sig);
        loadParams.push(Web3.utils.asciiToHex(message));
        selfweb3.GetWeb3().Web3Execute("call", "Load", walletAddress, 0, loadParams, function (loadResult) {
            selfweb3.ShowMsg('', flow, 'web3 contract: Web3Public successed: ', loadResult);
            let recoverID = Web3.utils.hexToAscii(loadResult['recoverID']);
            let web3Public = Web3.utils.hexToAscii(loadResult['web3Public']);
            selfweb3.ShowWaitting(false);
            selfweb3.SetProps("recoverID", recoverID);
            selfweb3.SetProps("web3Public", web3Public);
            if (callback !== undefined && callback !== null) callback();
        }, function (err) {
            selfweb3.ShowWaitting(false);
            selfweb3.ShowMsg('error', flow, 'sign failed: ', err);
        });
    })
}

function initBackend(flow, walletAddress, inputWeb2Key, callback) {
    let userID = walletAddress;
    WasmPublic(function(wasmResponse) {
        let queryMap = {};
        queryMap['userID'] = userID;
        queryMap['kind'] = "web2Data";
        queryMap['params'] = "initWeb2";
        queryMap['public'] = JSON.parse(wasmResponse)['Data'];
        selfweb3.SetProps('wasmPublic', JSON.parse(wasmResponse)['Data']);
        selfweb3.httpGet("/api/datas/load", queryMap, function(response) {
            if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
                selfweb3.ShowMsg('error', flow, 'init web2 service failed: ', response['Error']);
            } else {
                let web2Response = response['Data'];
                WasmInit(walletAddress, inputWeb2Key, web2Response['Web2NetPublic'], web2Response['Web2Data'], function(initResponse) {
                    let wasmResp = {};
                    wasmResp = JSON.parse(initResponse);
                    if (wasmResp['Error'] !== '' && wasmResp['Error'] !== null && wasmResp['Error'] !== undefined) {
                        selfweb3.wasmCallback("WasmInit", response['Error'], false);
                    } else {
                        if (callback !== undefined && callback !== null) callback(wasmResp['Data'], web2Response['Web2Address']);
                        // initWeb3(flow, walletAddress, wasmResp['Data'], web2Response['Web2Address'], callback);
                    }
                });
            }
        })
    });
}

/* example
    import * as selfweb3 from '../logic/index.js';
    selfweb3.GetUser().Register(selfAddress, walletAddress, email, function(qrcode){
        self.showQRcode(qrcode);
        setTimeout(function() {
            self.qrcodeUrl = '';
            window.location.reload();
        }, 60000);
    })
*/
export function Register(selfAddress, walletAddress, recoverID, callback) {
    // wasm
    let userID = walletAddress;
    selfweb3.ShowWaitting(true);
    WasmRegister(userID, recoverID, function(wasmResponse) {
        let response = JSON.parse(wasmResponse);
        if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
            selfweb3.wasmCallback("WasmRegister", response['Error'], false);
        } else {
            selfweb3.wasmCallback("Register");
            var registParams = [];
            registParams.push(selfAddress);
            registParams.push(Web3.utils.asciiToHex(response['Data']['RecoverID']));
            registParams.push(Web3.utils.asciiToHex(response['Data']['Web3Key']));
            registParams.push(Web3.utils.asciiToHex(response['Data']['Web3Public']));
            // 流程: contract.Register ===> webAuthnRegister ===> /api/datas/store ===> TOTP QRCode
            selfweb3.GetWeb3().Web3Execute("send", "Register", walletAddress, 0, registParams, function (result) {
                selfweb3.ShowWaitting(false);

                // webAuthn register
                // self.$parent.getSelf().$refs.webauthn.webRegister(userID, function(){
                //     self.$parent.getSelf().enableSpin(false);
                //     self.storeWeb2Data(userID, recoverID, response['Data']['Web2Data'], response['Data']['QRCode']);
                // }, function() {
                //     self.$parent.getSelf().enableSpin(false);
                //     self.$Message.error('webAuthn register failed');
                // });

                StoreSelfData(userID, recoverID, response['Data']['Web2Data'], function() {
                    if (callback !== undefined && callback !== null) callback(response['Data']['QRCode']);
                });
            }, function (err) {
                selfweb3.ShowWaitting(false);
                selfweb3.ShowMsg('error', 'Register', 'web3 contract: register failed: ', walletAddress);
            })
        }
    })
}

export function StoreSelfData(userID, recoverID, web2Data, callback) {
    let formdata = new FormData();
    formdata.append("userID", userID);
    formdata.append("kind", 'web2Data');
    formdata.append("params", web2Data);
    formdata.append("recoverID", recoverID);
    selfweb3.httpPost("/api/datas/store", formdata, function(storeResponse) {
        if (storeResponse['Error'] == '') {
            if (callback !== undefined && callback !== null) callback();
        } else {
            selfweb3.ShowMsg('error', 'StoreSelfData', 'store selfData failed: ', storeResponse['Error']);
        }
    })
}

// export function EmailVerify(code, emailMap, callback) {
//     let userID = self.$parent.getSelf().getWalletAddress();
//     WasmVerify(userID, code, 'email', JSON.stringify(emailMap), function(wasmResponse) {
//         let response = JSON.parse(wasmResponse);
//         console.log('emailVerify: ', response);
//         if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
//             selfweb3.wasmCallback("WasmVerify", response['Error'], false);
//         } else {
//             if (callback !== undefined && callback !== null) callback(response['Data']);
//         }
//     })
// }

// export function TOTPVerify(code, emailMap, callback) {

// }

// export function WebAuthnVerify(code, emailMap, callback) {

// }

// // 关联验证流程: web3合约: 钱包签名校验(提取web3Public) ---> TOTP校验 ---> web3合约: zk证明校验, 钱包签名校验(提取web3Key) ---> webAuthn登录校验
// export function RelationVerify(bTOTP, bWebAuthn, callback, failed) {
//     let self = this;
//     let userID = self.$parent.getSelf().getWalletAddress();
//     let verifyWebAuthn = function(params) {
//         let loadParams = [];
//         loadParams.push(self.$parent.getSelf().selfAddress);
//         loadParams = loadParams.concat(params);
//         self.$parent.getSelf().$refs.walletPanel.Execute("call", "Web3Key", userID, 0, loadParams, function (loadResult) {
//             console.log('web3 contract: Load from contract successed: ', loadResult);
//             let web3Map = {"method": "WebAuthnKey", "web3Key": Web3.utils.hexToAscii(loadResult), "web3Public": self.web3Public};
//             WasmHandle(userID, JSON.stringify(web3Map), function(wasmWebAuthnResponse) {
//                 self.$parent.getSelf().$refs.webauthn.webLogin(self.$parent.getSelf().getWalletAddress(), JSON.parse(wasmWebAuthnResponse)['Data'], function() {
//                     if (callback !== undefined && callback !== null) callback();
//                 }, function() {
//                     if (failed !== undefined && failed !== null) failed();
//                 })
//             })
//         }, function (err) {
//             self.$Message.error('web3 contract: Load from contract failed');
//             if (failed !== undefined && failed !== null) failed();
//         })
//     }

//     if (bTOTP === true) {
//         let relateTimes = "1";
//         let web3Map = {"method": "RelationVerify", "web3Key": '', "web3Public": self.web3Public, "action": "dapp"};
//         self.$parent.getSelf().switchPanel('RelationVerify', '', JSON.stringify(web3Map), function(wasmTOTPResponse){
//             if (bWebAuthn === true) {
//                 packRelateVerifyParams(web3Map['action'], wasmTOTPResponse, verifyWebAuthn);
//             } else {
//                 if (callback !== undefined && callback !== null) callback(wasmTOTPResponse);
//             }
//         })
//     } else if (bWebAuthn === true) {
//         verifyWebAuthn([]);
//     }
// }

// // action: dapp, deposit, withdraw
// function packRelateVerifyParams(action, verifyParams, callback) {
//     console.log('packRelateVerifyParams: ', action, verifyParams);
//     let self = this;
//     let queryMap = {};
//     queryMap['action'] = action;
//     queryMap['kind'] = 'relateVerify';
//     queryMap['nonce'] = verifyParams['nonce'];
//     queryMap['userID'] = self.$parent.getSelf().getWalletAddress();
//     self.$parent.getSelf().httpGet("/api/datas/load", queryMap, function(response) {
//         if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
//             self.$Message.error('load datas from web2 service failed: ', response.data['Error']);
//         } else {
//             const relateVerifyParams = response.data['Data'];
//             console.log('before packRelateVerifyParams: ', relateVerifyParams);
//             const merkleParams = self.$parent.getSelf().$refs.walletPanel.getWeb3().eth.abi.encodeParameter(
//                 {
//                     "VerifyParam": {
//                         "kindList": 'uint256[]',
//                         "msgList": 'bytes[]',
//                         "sigList": 'bytes[]',
//                         "proofs": 'bytes32[][]',
//                         "leaves": 'bytes32[]',
//                     }
//                 },
//                 {
//                     "kindList": [1, 2],
//                     "msgList": [Web3.utils.asciiToHex(relateVerifyParams['message']), Web3.utils.asciiToHex(verifyParams['message'])],
//                     "sigList": [relateVerifyParams['signature'], verifyParams['signature']],
//                     "proofs": [relateVerifyParams['proofs']],
//                     "leaves": relateVerifyParams['leaves'],
//                 }
//             )
//             console.log('&^^^^^^^^^^^^^^^^^merkleParams: ', merkleParams, relateVerifyParams, verifyParams);
//             if (callback !== undefined && callback !== null) callback(merkleParams);
//         }
//     })
// }