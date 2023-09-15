// 处理所有用户相关的业务逻辑, 包括Load, Register, Recover, Reset等
"use strict"
import Web3 from "web3";
import * as verify3 from './verify.js';
import * as selfweb3 from './index.js';

export const Flow_Init = "Init";
export const Flow_Load = "Load";
export const Flow_Reset = "Reset";
export const Flow_Register = "Register";
export const Flow_EnterDapp = "EnterDapp";
export const Flow_Registered = "Registered";
export const Flow_StoreSelfData = "StoreSelfData";

// callback(selfAddress, web2Address)
export function Init(walletAddress, inputWeb2Key, callback) {
    initBackend(Flow_Init, walletAddress, inputWeb2Key, callback);
}

// callback(registered, bound)
export function Registered(walletAddress, selfAddress, callback) {
    selfweb3.GetWeb3().Execute("call", "Registered", walletAddress, 0, [selfAddress], function (loadResult) {
        selfweb3.ShowMsg('', Flow_Registered, 'web3 Registered successed', '');
        if (callback !== undefined && callback !== null) callback(loadResult, true);
    }, function (err) {
        selfweb3.ShowMsg('error', Flow_Registered, 'web3 contract call failed', err);
    });
}

// callback()
export function Load(walletAddress, selfAddress, callback) {
    initWeb3(Flow_Load, walletAddress, selfAddress, callback);
}

function initWeb3(flow, walletAddress, selfAddress, callback) {
    let message = 'SelfWeb3 Init: ' + (new Date()).getTime();
    selfweb3.GetWeb3().Sign(walletAddress, message, function(sig) {
        var loadParams = [];
        loadParams.push(selfAddress);
        loadParams.push(sig);
        loadParams.push(Web3.utils.asciiToHex(message));
        selfweb3.GetWeb3().Execute("call", "Load", walletAddress, 0, loadParams, function (loadResult) {
            let recoverID = Web3.utils.hexToAscii(loadResult['recoverID']);
            let web3Public = Web3.utils.hexToAscii(loadResult['web3Public']);
            selfweb3.SetProps("recoverID", recoverID);
            selfweb3.SetProps("web3Public", web3Public);
            selfweb3.ShowMsg('', flow, 'user load successed', [recoverID, web3Public]);
            if (callback !== undefined && callback !== null) callback();
        }, function (err) {
            selfweb3.ShowMsg('error', flow, 'sign message failed', err);
        });
    }, function(err) {
        selfweb3.ShowMsg('error', flow, 'sign message failed', err);
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
                selfweb3.ShowMsg('error', flow, 'init web2 server failed', response['Error']);
            } else {
                let web2Response = response['Data'];
                selfweb3.ShowMsg("", flow, "inint web2 server successed", web2Response);
                WasmInit(walletAddress, inputWeb2Key, web2Response['Web2NetPublic'], web2Response['Web2Data'], function(initResponse) {
                    let wasmResp = {};
                    wasmResp = JSON.parse(initResponse);
                    if (wasmResp['Error'] !== '' && wasmResp['Error'] !== null && wasmResp['Error'] !== undefined) {
                        selfweb3.ShowMsg("error", flow, "inint web2 server failed", response['Error']);
                    } else {
                        selfweb3.ShowMsg("", flow, "inint wasm successed", [wasmResp['Data'], web2Response['Web2Address']]);
                        if (callback !== undefined && callback !== null) callback(wasmResp['Data'], web2Response['Web2Address']);
                    }
                });
            }
        }, function(err) {
            selfweb3.ShowMsg("error", flow, "inint web2 server failed", err);
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

    // callback(qrcode)
*/
export function Register(walletAddress, selfAddress, recoverID, callback) {
    // wasm
    let userID = walletAddress;
    WasmRegister(userID, recoverID, function(wasmResponse) {
        let response = JSON.parse(wasmResponse);
        if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
            selfweb3.ShowMsg("error", Flow_Register, "user register failed", response['Error']);
        } else {
            var registParams = [];
            registParams.push(selfAddress);
            registParams.push(Web3.utils.asciiToHex(response['Data']['RecoverID']));
            registParams.push(Web3.utils.asciiToHex(response['Data']['Web3Key']));
            registParams.push(Web3.utils.asciiToHex(response['Data']['Web3Public']));
            // 流程: contract.Register ===> webAuthnRegister ===> /api/datas/store ===> TOTP QRCode
            selfweb3.GetWeb3().Execute("send", Flow_Register, walletAddress, 0, registParams, function (result) {
                // webAuthn register
                verify3.WebAuthnRegister(Flow_Register, userID, function(){
                    StoreSelfData(userID, recoverID, response['Data']['Web2Data'], function() {
                        selfweb3.ShowMsg('', Flow_Register, 'user register successed', selfAddress);
                        if (callback !== undefined && callback !== null) callback(response['Data']['QRCode']);
                    });
                }, function() {
                    selfweb3.ShowMsg('error', Flow_Register, 'user register failed', err);
                });
            }, function (err) {
                selfweb3.ShowMsg('error', Flow_Register, 'user register failed', err);
            })
        }
    })
}

// callback()
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
            selfweb3.ShowMsg('error', Flow_StoreSelfData, 'store selfData failed', storeResponse['Error']);
        }
    })
}

// callback(resetParams)
export function Reset(walletAddress, selfAddress, code, resetKind, callback) {
    if (resetKind === "TOTP") {
        let verifyParams = {"method": "ResetTOTPKey", "recoverID": selfweb3.GetProps('recoverID'), "web3Public": selfweb3.GetProps('web3Public'), "action": "query"};
        verify3.FinishEmailVerify(walletAddress, code, JSON.stringify(verifyParams), function(wasmResponse){
            packRelateVerifyParams(Flow_Reset, walletAddress, 'query', wasmResponse['Data']['RelateVerify'], function(merkleParams){
                verify3.WebAuthnVerify(Flow_Reset, walletAddress, selfAddress, merkleParams, function(){
                    StoreSelfData(walletAddress, '', wasmResponse['Data']['Reset']['Web2Data'], function() {
                        if (callback !== undefined && callback !== null) callback(wasmResponse['Data']['Reset']['QRCode']);
                    });
                });
            });
        });
    } else if (resetKind === "Web2Key") {
        console.log(resetKind)
    } else if (resetKind === "WebAuthn") {
        console.log(resetKind)
    }
}

/*
1. beginTOTPVerify:
    function beginTOTPVerify(callback) {
        // enable TOTP code input
        // callback(code)
    }

2. callback()
*/
export function EnterDapp(walletAddress, selfAddress, beginTOTPVerify, callback) {
    if (beginTOTPVerify === undefined || beginTOTPVerify === null) {
        selfweb3.ShowMsg("error", Flow_EnterDapp, "load web3Key failed", "invalid beginTOTPVerify function");
        return;
    }
    console.log("EnterDapp: ", walletAddress, selfAddress)

    beginTOTPVerify(function(code){
        let web3Map = {"method": "RelationVerify", "web3Key": '', "web3Public": selfweb3.GetProps('web3Public'), "action": "query"};
        verify3.TOTPVerify(Flow_EnterDapp, walletAddress, code, JSON.stringify(web3Map), function(wasmResponse) {
            packRelateVerifyParams(Flow_EnterDapp, walletAddress, web3Map['action'], wasmResponse, function(merkleParams){
                verify3.WebAuthnVerify(Flow_EnterDapp, walletAddress, selfAddress, merkleParams, callback);
            });
        })
    })
}

// action: query, update
function packRelateVerifyParams(flow, walletAddress, action, verifyParams, callback) {
    console.log('packRelateVerifyParams: ', action, verifyParams);
    let queryMap = {};
    queryMap['action'] = action;
    queryMap['kind'] = 'relateVerify';
    queryMap['nonce'] = verifyParams['nonce'];
    queryMap['userID'] = walletAddress;
    selfweb3.httpGet("/api/datas/load", queryMap, function(response) {
        if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
            selfweb3.ShowMsg('error', flow, 'load datas from web2 server failed', response['Error']);
        } else {
            const relateVerifyParams = response['Data'];
            console.log('before packRelateVerifyParams: ', relateVerifyParams);
            console.log(Web3.utils.asciiToHex(relateVerifyParams['message']))
            console.log(Web3.utils.asciiToHex(verifyParams['message']))
            
            const merkleParams = selfweb3.GetWeb3().web3.eth.abi.encodeParameter(
                {
                    "VerifyParam": {
                        "kindList": 'uint256[]',
                        "msgList": 'bytes[]',
                        "sigList": 'bytes[]',
                        "proofs": 'bytes32[][]',
                        "leaves": 'bytes32[]',
                    }
                },
                {
                    "kindList": [1, 2],
                    "msgList": [Web3.utils.asciiToHex(relateVerifyParams['message']), Web3.utils.asciiToHex(verifyParams['message'])],
                    "sigList": [relateVerifyParams['signature'], verifyParams['signature']],
                    "proofs": [relateVerifyParams['proofs']],
                    "leaves": relateVerifyParams['leaves'],
                }
            )
            console.log('&^^^^^^^^^^^^^^^^^merkleParams: ', merkleParams, relateVerifyParams, verifyParams);
            if (callback !== undefined && callback !== null) callback(merkleParams);
        }
    })
}
