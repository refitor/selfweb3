"use strict"
import * as selfweb3 from './index.js';

export const Flow_TOTPVerify = "TOTPVerify";
export const Flow_WebAuthnVerify = "WebAuthnVerify";
export const Flow_BeginEmailVerify = "BeginEmailVerify";
export const Flow_FinishEmailVerify = "FinishEmailVerify";

// callback()
export function BeginEmailVerify(walletAddress, email, callback) {
    let userID = walletAddress;
    WasmAuthorizeCode(userID, email, function(wasmResponse) {
        let response = {};
        response = JSON.parse(wasmResponse);
        if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
            selfweb3.ShowMsg("error", Flow_BeginEmailVerify, "email push failed: ", response['Error']);
        } else {
            let formdata = new FormData();
            formdata.append("userID", userID);
            formdata.append("kind", 'email');
            formdata.append("params", response['Data']);
            formdata.append("public", selfweb3.GetProps('wasmPublic'));
            selfweb3.httpPost(selfweb3.GetProps('ApiPrefix') + "/api/datas/forward", formdata, function(forwardResponse) {
                if (forwardResponse['Error'] == '') {
                    if (callback !== undefined && callback !== null) callback();
                } else {
                    selfweb3.ShowMsg("error", Flow_BeginEmailVerify, "email push failed: ", forwardResponse['Error']);
                }
            })
        }
    })
}

export function FinishEmailVerify(walletAddress, code, verifyParams, callback) {
    WasmVerify(walletAddress, code, 'Email', verifyParams, function(wasmResponse) {
        let response = JSON.parse(wasmResponse);
        if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
            selfweb3.ShowMsg("error", Flow_FinishEmailVerify, "email verify failed", response['Error']);
        } else {
            console.log("FinishEmailVerify successed, response: ", response);
            if (callback !== undefined && callback !== null) callback(response);
        }
    })
}

export function TOTPVerify(flow, walletAddress, code, verifyParams, callback) {
    console.log("TOTPVerify: ", flow, walletAddress, code, verifyParams)
    WasmVerify(walletAddress, code, 'TOTP', verifyParams, function(wasmResponse) {
        console.log("TOTPVerify wasmResponse: ", wasmResponse)
        let response = JSON.parse(wasmResponse);
        if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
           selfweb3.ShowMsg('error', flow, 'TOTP verify failed', response['Error']);
        } else {
            if (callback !== undefined && callback !== null) callback(response['Data']);
        }
    })
}

export function WebAuthnRegister(flow, userID, callback, failed) {
    let handleFailed = function(err) {
        selfweb3.ShowMsg("error", flow, ' webAuthn register failed', err);
        if (failed !== undefined && failed !== null) failed(err);
    }

    // let name = walletAddress; //"wallet-" + walletAddress.substring(0, 4) + "..." + walletAddress.substring(walletAddress.length - 4, walletAddress.length);
    let formdata = new FormData();
    formdata.append('userID', userID);
    fetch(selfweb3.GetProps('ApiPrefix') + '/api/user/begin/register', {
        method: 'POST',
        body: formdata,
    })
    .then(selfweb3.checkStatus(200))
    .then(res => selfweb3.checkError(res, handleFailed))
    .then(response => {
        console.log("+++++++++++++", response)
        let credentialCreationOptions = response["Data"];
        credentialCreationOptions.publicKey.challenge = bufferDecode(credentialCreationOptions.publicKey.challenge);
        credentialCreationOptions.publicKey.user.id = bufferDecode(credentialCreationOptions.publicKey.user.id);
        if (credentialCreationOptions.publicKey.excludeCredentials) {
            for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
            credentialCreationOptions.publicKey.excludeCredentials[i].id = bufferDecode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
            }
        }
        console.log('=================111: ', credentialCreationOptions)
        return credentialCreationOptions;
    })
    .then(credentialCreationOptions => navigator.credentials.create(credentialCreationOptions))
    .then(credential => {
        console.log(credential)
        let attestationObject = credential.response.attestationObject;
        let clientDataJSON = credential.response.clientDataJSON;
        let rawId = credential.rawId;

        console.log('=================222: ', credential, "++++", clientDataJSON, attestationObject)
        fetch(selfweb3.GetProps('ApiPrefix') + '/api/user/finish/register?userID=' + userID, {
            method: 'POST',
            headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: credential.id,
                rawId: bufferEncode(rawId),
                type: credential.type,
                response: {
                    attestationObject: bufferEncode(attestationObject),
                    clientDataJSON: bufferEncode(clientDataJSON),
                },
            }),
        })
        .then(selfweb3.checkStatus(200))
        .then(res => selfweb3.checkError(res, handleFailed))
        .then(response => {
            if (callback !== undefined && callback !== null) callback();
        })
    })
}

export function WebAuthnLogin(flow, userID, webAuthnKey, callback, failed) {
    let handleFailed = function(err) {
        selfweb3.ShowMsg("error", flow, 'webAuthn login failed', err);
        if (failed !== undefined && failed !== null) failed(err);
    }

    let formdata = new FormData();
    formdata.append('userID', userID);
    formdata.append('webAuthnKey', webAuthnKey);
    fetch(selfweb3.GetProps('ApiPrefix') + '/api/user/begin/login', {
        method: 'POST',
        body: formdata,
    })
    .then(selfweb3.checkStatus(200))
    .then(res => selfweb3.checkError(res, handleFailed))
    .then(response => {
        let credentialRequestOptions = response["Data"];
        console.log('start=================333: ', credentialRequestOptions)
        credentialRequestOptions.publicKey.challenge = bufferDecode(credentialRequestOptions.publicKey.challenge);
        credentialRequestOptions.publicKey.allowCredentials.forEach(function (listItem) {
            listItem.id = bufferDecode(listItem.id)
        });
        console.log('=================333: ', credentialRequestOptions)
        return credentialRequestOptions;
    })
    .then(credentialRequestOptions => navigator.credentials.get({publicKey: credentialRequestOptions.publicKey}))
    .then(assertion => {
        let authData = assertion.response.authenticatorData;
        let clientDataJSON = assertion.response.clientDataJSON;
        let rawId = assertion.rawId;
        let sig = assertion.response.signature;
        let userHandle = assertion.response.userHandle;

        console.log('=================444: ', assertion)
        fetch(selfweb3.GetProps('ApiPrefix') + '/api/user/finish/login?userID=' + userID, {
            method: 'POST',
            headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                id: assertion.id,
                rawId: bufferEncode(rawId),
                type: assertion.type,
                response: {
                    authenticatorData: bufferEncode(authData),
                    clientDataJSON: bufferEncode(clientDataJSON),
                    signature: bufferEncode(sig),
                    userHandle: bufferEncode(userHandle),
                },
            }),
        })
        .then(selfweb3.checkStatus(200))
        .then(res => selfweb3.checkError(res, handleFailed))
        .then(response => {
            if (callback !== undefined && callback !== null) callback();
        })
    })
}

// Base64 to ArrayBuffer
function bufferDecode(value) {
    return Uint8Array.from(atob(value), c => c.charCodeAt(0));
}

// ArrayBuffer to URLBase64
function bufferEncode(value) {
    return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");;
}