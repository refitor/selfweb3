<template>

</template>

<script>
// import * as CBOR from "cbor-web";
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
        }
    }
}
</script>