// 作为全局组件处理dapp相关的业务逻辑, 提供所有与web3, web2以及wasm交互相关的接口(全局性), 再按业务细分，用户相关的user.js, web3相关的web3.js
// dapp.js Function: Init, RelateVerify, RunDapp等
// user.js Function: Load, Register, Recover, Reset等
// web3.js Function: Web3Init, Web3Params, Web3Execute等
"use strict"
import * as wasm from "./wasm.js";
import * as user from "./user.js";
import * as web3 from "./web3.js";
import * as verify3 from "./verify.js";

/* 使用方式: 
    // 导入index.js即可
    import * as selfweb3 from './logic/index.js';

    let showMsg = function(err, flow, msg, param) {
        console.log({'flow': flow, 'error': err === 'error', 'msg': msg, 'param': param});
    }

    // 初始化js库
    let walletAddress = this.walletAddress;
    selfweb3.Init(selfweb3.GetWeb3().ContractSelfWeb3, this.$refs.walletPanel.web3.currentProvider, showMsg, function() {
        // inputWeb2Key: 用户自己输入web2服务密钥, 可先为空
        selfweb3.GetUser().Init(walletAddress, '', function(selfAddress, web2Address) {
            // check registered
            selfweb3.GetUser().Registered(walletAddress, selfAddress, function(registered, bound){
                if (registered === true) {
                    if (bound === true) {
                        // 已注册, 钱包地址一致, 开始加载用户私有信息
                        selfweb3.GetUser().Load(walletAddress, selfAddress, function(){
                            console.log('// 已注册, 钱包地址一致, 用拿到的地址信息初始化profile(第一个卡片的内容), 用户加载流程完成')
                        });
                    } else {
                        console.log('// 已注册, 但钱包地址不一致, 弹出modal框提示是否重新绑定钱包, 启动钱包重新绑定流程')
                    }
                } else {
                    console.log('// 尚未注册')
                }
            })
        }, function(err){
            console.log('// 已注册, 钱包地址一致, 但需要用户自行输入web2服务密钥解密私有数据, 弹出modal框提示用户输入web2服务密钥, 确认后重新走selfweb3.GetUser().Init流程')
        })
    });

    // 如果有必要, 页面销毁前清空js中的变量存储
    selfweb3.UnInit()
*/

// err: 如果是错误消息, 值为error, 如果是正常的消息, 值为空字符串
// flow: 流程名称, 比如Init, Load, Register, Reset等
// param: 可选的业务数据参数
export let ShowMsg = function(err, flow, msg, param) {
    console.log("error: ", err === 'error', flow, msg, param);
}

// 可以调整到外部做存储管理
let Props = {};

export function GetProps(key) {
    return Props[key];
}

export function SetProps(key, val) {
    Props[key] = val;
}

// callback: function()
export function Init(contractName, provider, showMsg, callback) {
    UnInit();
    if (showMsg !== null && showMsg !== undefined) ShowMsg = showMsg;

    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/selfweb3.wasm"), go.importObject)
    .then(function(result) {
        go.run(result.instance);
        web3.Init(contractName, provider, callback)
    })
}

export function UnInit() {
    Props = {};
    web3.UnInit();
}

export function GetWeb3() {
    return web3;
}

export function GetUser() {
    return user;
}

export function GetVerify() {
    return verify3;
}

export function wasmCallback(flow, method, err, spinStatus) {
    if (spinStatus !== undefined) ShowWaitting(spinStatus);
    if (err === undefined || err === '') {
        ShowMsg('', flow, 'exec wasm method successed: ', method);
    } else {
        ShowMsg('error', flow, 'exec wasm method failed: ', method + ", " + err);
    }
}

export function httpGet(url, params, callback, failed) {
    let handleFailed = function(err) {
        ShowMsg("error", 'httpGet', 'GET ' + url + ' failed', '');
        if (failed !== undefined && failed !== null) failed(response['Error']);
    }

    fetch(url + "?" + new URLSearchParams(params).toString())
    .then(checkStatus(200))
    .then(res => checkError(res, handleFailed))
    .then((response) => {
        if (callback !== undefined && callback !== null) callback(response);
    })
}

export function httpPost(url, formdata, callback, failed) {
    let handleFailed = function(err) {
        ShowMsg("error", 'httpGet', 'POST ' + url + ' failed', '');
        if (failed !== undefined && failed !== null) failed(response['Error']);
    }

    fetch(url, {
        method: 'POST',
        body: formdata,
    })
    .then(checkStatus(200))
    .then(res => checkError(res, handleFailed))
    .then((response) => {
        if (callback !== undefined && callback !== null) callback(response);
    })
}

export function checkStatus(status) {
    return res => {
        if (res.status === status) {
            return res.json();
        }
        throw new Error(res.statusText);
    };
}

export function checkError(response, failed) {
    if (response['Error'] === '') {
        return response;
    }
    if (failed !== undefined && failed !== null) failed(response['Error']);
    throw new Error(response['Error']);
}