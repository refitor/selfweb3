<template>

</template>

<script>
import * as CBOR from "cbor-web";
export default {
    mounted: function () {
    },
    methods: {
        register(walletAddress, callback) {
            let self = this;
            let response = {};
            let name = "wallet-" + walletAddress.substring(0, 4) + "..." + walletAddress.substring(walletAddress.length - 4, walletAddress.length);
            console.log('before BeginRegister: ', walletAddress, name);
            BeginRegister(walletAddress, name, name, async function(wasmResponse){
                const response = JSON.parse(wasmResponse);
                let responseData = response['Data'];
                if (responseData !== null && responseData !== undefined && responseData !== {}) {
                    console.log('after BeginRegister: ', responseData);
                    responseData.publicKey.challenge = Uint8Array.from("RmM1dmc3ZEp4MjZ0ZmVrdUFHLXNOU0VFVUo0Q0dyaW9GVUp3bGVOaWx5NA", c => c.charCodeAt(0));
                    responseData.publicKey.user.id = Uint8Array.from(responseData.publicKey.user.id, c => c.charCodeAt(0));

                    const utf8Decoder = new TextDecoder('utf-8');
                    let credential = await navigator.credentials.create(responseData);
                    const decodedClientData = utf8Decoder.decode(credential.response.clientDataJSON);

                    console.log('before FinishRegister: ', decodedClientData);
                    FinishRegister(walletAddress, decodedClientData, function(wasmResponse) {
                        console.log('after FinishRegister: ', JSON.parse(wasmResponse));
                        if (callback !== undefined && callback !== null) callback();
                    })
                } else {
                    if (callback !== undefined && callback !== null) callback(response['Error']);
                }
            })
        },
        login(walletAddress, callback) {
            let self = this;
            console.log('before LoginRegister: ', walletAddress);
            BeginLogin(walletAddress, async function(wasmResponse) {
                const response = JSON.parse(wasmResponse);
                let responseData = response['Data'];
                console.log('after BeginLogin: ', responseData);
                if (responseData !== null && responseData !== undefined && responseData !== {}) {
                    console.log('before FinishLogin: ', responseData);
                    
                    const publicKeyCredentialRequestOptions = {
                        challenge: Uint8Array.from(
                            responseData.publicKey.challenge, c => c.charCodeAt(0)),
                        allowCredentials: [{
                            id: Uint8Array.from(
                                walletAddress, c => c.charCodeAt(0)),
                            type: 'public-key',
                            transports: ['hybrid'],
                        }],
                        timeout: 60000,
                    }
                    responseData.publicKey = publicKeyCredentialRequestOptions
                    // responseData.publicKey.challenge = Uint8Array.from(responseData.publicKey.challenge, c => c.charCodeAt(0));
                    // responseData.publicKey.user.id = Uint8Array.from(responseData.publicKey.user.id, c => c.charCodeAt(0));

                    const utf8Decoder = new TextDecoder('utf-8');
                    let credential = await navigator.credentials.get(responseData);
                    const decodedClientData = utf8Decoder.decode(credential.response.clientDataJSON);

                    FinishLogin(walletAddress, decodedClientData, function(wasmResponse) {
                        console.log('after FinishLogin: ', JSON.parse(wasmResponse));
                        if (callback !== undefined && callback !== null) callback();
                    })
                } else {
                    if (callback !== undefined && callback !== null) callback(response['Error']);
                }
            })
        },
        webRegister(userID, callback, failed) {
            let self = this;
            // let name = walletAddress; //"wallet-" + walletAddress.substring(0, 4) + "..." + walletAddress.substring(walletAddress.length - 4, walletAddress.length);
            let formdata = new FormData();
            formdata.append('userID', userID);
            fetch('/api/user/begin/register', {
				method: 'POST',
                body: formdata,
			})
			.then(self.checkStatus(200))
            .then(res => self.checkError(res, failed))
			.then(response => {
                console.log("+++++++++++++", response)
                let credentialCreationOptions = response["Data"];
                credentialCreationOptions.publicKey.challenge = self.bufferDecode(credentialCreationOptions.publicKey.challenge);
                credentialCreationOptions.publicKey.user.id = self.bufferDecode(credentialCreationOptions.publicKey.user.id);
                if (credentialCreationOptions.publicKey.excludeCredentials) {
                    for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
                    credentialCreationOptions.publicKey.excludeCredentials[i].id = self.bufferDecode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
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
				fetch('/api/user/finish/register?userID=' + userID, {
					method: 'POST',
					headers: {
						'Accept': 'application/json',
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({
                        id: credential.id,
                        rawId: self.bufferEncode(rawId),
                        type: credential.type,
                        response: {
                            attestationObject: self.bufferEncode(attestationObject),
                            clientDataJSON: self.bufferEncode(clientDataJSON),
                        },
                    }),
				})
                .then(self.checkStatus(200))
                .then(res => self.checkError(res, failed))
                .then(response => {
                    if (callback !== undefined && callback !== null) callback();
                })
			})
        },
        webLogin(userID, webAuthnKey, callback, failed) {
            let self = this;
            let formdata = new FormData();
            formdata.append('userID', userID);
            formdata.append('webAuthnKey', webAuthnKey);
            fetch('/api/user/begin/login', {
				method: 'POST',
                body: formdata,
			})
			.then(self.checkStatus(200))
            .then(res => self.checkError(res, failed))
            .then(response => {
                let credentialRequestOptions = response["Data"];
                console.log('start=================333: ', credentialRequestOptions)
                credentialRequestOptions.publicKey.challenge = self.bufferDecode(credentialRequestOptions.publicKey.challenge);
                credentialRequestOptions.publicKey.allowCredentials.forEach(function (listItem) {
                    listItem.id = self.bufferDecode(listItem.id)
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
				fetch('/api/user/finish/login?userID=' + userID, {
					method: 'POST',
					headers: {
						'Accept': 'application/json',
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({
                        id: assertion.id,
                        rawId: self.bufferEncode(rawId),
                        type: assertion.type,
                        response: {
                            authenticatorData: self.bufferEncode(authData),
                            clientDataJSON: self.bufferEncode(clientDataJSON),
                            signature: self.bufferEncode(sig),
                            userHandle: self.bufferEncode(userHandle),
                        },
                    }),
				})
                .then(self.checkStatus(200))
                .then(res => self.checkError(res, failed))
                .then(response => {
                    if (callback !== undefined && callback !== null) callback();
                })
			})
        },
        // Decode a base64 string into a Uint8Array.
        _decodeBuffer(value) {
            // return Uint8Array.from(atob(value), c => c.charCodeAt(0));
            return Uint8Array.from(value, c => c.charCodeAt(0));
        },
        // Encode an ArrayBuffer into a base64 string.
        _encodeBuffer(value) {
            return btoa(new Uint8Array(value).reduce((s, byte) => s + String.fromCharCode(byte), ''));
        },
        // Checks whether the status returned matches the status given.
        checkStatus(status) {
            return res => {
                if (res.status === status) {
                    return res.json();
                }
                throw new Error(res.statusText);
            };
        },
        checkError(response, failed) {
            if (response['Error'] === '') {
                return response;
            }
            console.log('checkError: ', response['Error']);
            if (failed !== undefined && failed !== null) failed();
        },
        // Base64 to ArrayBuffer
        bufferDecode(value) {
            return Uint8Array.from(atob(value), c => c.charCodeAt(0));
        },
        // ArrayBuffer to URLBase64
        bufferEncode(value) {
            return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
            .replace(/\+/g, "-")
            .replace(/\//g, "_")
            .replace(/=/g, "");;
        }
    }
}
</script>