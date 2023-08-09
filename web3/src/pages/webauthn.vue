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
        webRegister(walletAddress, callback) {
            let self = this;
            let name = walletAddress; //"wallet-" + walletAddress.substring(0, 4) + "..." + walletAddress.substring(walletAddress.length - 4, walletAddress.length);
            console.log('before BeginRegister: ', walletAddress, name);
        
            self.doRegister(walletAddress, name)
			.then(res => alert('This authenticator has been registered'))
			.catch(err => {
				console.error(err)
				alert('Failed to register: ' + err);
			})
			.then(() => {
				if (callback !== undefined && callback !== null) callback();
			});
        },
        webLogin(walletAddress, callback) {
            let self = this;
            let name = "wallet-" + walletAddress.substring(0, 4) + "..." + walletAddress.substring(walletAddress.length - 4, walletAddress.length);

            self.doLogin(walletAddress, name)
			.then(res => res.json())
			.then(res => alert('You have been logged in to ' + res.name))
			.catch(err => {
				console.error(err)
				alert('Failed to login: ' + err);
			})
			.then(() => {
                if (callback !== undefined && callback !== null) callback();
			});
        },
        doRegister(id, name) {
            let self = this;
            let formdata = new FormData();
            formdata.append('id', id);
            formdata.append('name', name);
            return fetch('/api/user/begin/register', {
				method: 'POST',
                body: formdata,
			})
			.then(self._checkStatus(200))
			.then(res => res.json())
			.then(res => {
				res.publicKey.challenge = self._decodeBuffer(res.publicKey.challenge);
				res.publicKey.user.id = self._decodeBuffer(res.publicKey.user.id);
				if (res.publicKey.excludeCredentials) {
					for (var i = 0; i < res.publicKey.excludeCredentials.length; i++) {
						res.publicKey.excludeCredentials[i].id = self._decodeBuffer(res.publicKey.excludeCredentials[i].id);
					}
				}
                console.log('=================111: ', res)
				return res;
			})
			.then(res => navigator.credentials.create(res))
			.then(credential => {
                const utf8Decoder = new TextDecoder('utf-8');
                const decodedClientData = JSON.parse(utf8Decoder.decode(credential.response.clientDataJSON));
                const decodedAttestationObj = CBOR.decode(credential.response.attestationObject);

                const bodyBuf = JSON.stringify({
                    id: credential.id,
                    rawId: self._encodeBuffer(credential.rawId),
                    response: {
                        attestationObject: self._encodeBuffer(credential.response.attestationObject),
						clientDataJSON: self._encodeBuffer(credential.response.clientDataJSON)
                    },
                    type: credential.type
                })


                console.log('=================222: ', credential, "---", bodyBuf, "++++", decodedClientData, decodedAttestationObj)
				return fetch('/api/user/finish/register?name=' + name, {
					method: 'POST',
					headers: {
						'Accept': 'application/json',
						'Content-Type': 'application/json'
					},
					body: bodyBuf,
				})
			})
			.then(self._checkStatus(201));
        },
        doLogin(id, name) {
            let self = this;
            let formdata = new FormData();
            // formdata.append('id', id);
            formdata.append('name', name);
            return fetch('/api/user/begin/login', {
				method: 'POST',
                body: formdata,
			})
			.then(self._checkStatus(200))
			.then(res => res.json())
			.then(res => {
				res.publicKey.challenge = self._decodeBuffer(res.publicKey.challenge);
				if (res.publicKey.allowCredentials) {
					for (let i = 0; i < res.publicKey.allowCredentials.length; i++) {
						res.publicKey.allowCredentials[i].id = self._decodeBuffer(res.publicKey.allowCredentials[i].id);
					}
				}
                console.log('=================333: ', res)
				return res;
			})
			.then(res => navigator.credentials.get(res))
			.then(credential => {
                const bodyBuf = JSON.stringify({
                    id: credential.id,
						rawId: self._encodeBuffer(credential.rawId),
						response: {
							clientDataJSON: self._encodeBuffer(credential.response.clientDataJSON),
							authenticatorData: self._encodeBuffer(credential.response.authenticatorData),
							signature: self._encodeBuffer(credential.response.signature),
							userHandle: self._encodeBuffer(credential.response.userHandle),
						},
						type: credential.type
                })

                console.log('=================444: ', credential, bodyBuf)
				return fetch('/api/user/finish/login?name=' + name, {
					method: 'POST',
					headers: {
						'Accept': 'application/json',
						'Content-Type': 'application/json'
					},
					body: bodyBuf,
				})
			})
			.then(self._checkStatus(200));
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
        _checkStatus(status) {
            return res => {
                if (res.status === status) {
                    return res;
                }
                throw new Error(res.statusText);
            };
        }
    }
}
</script>