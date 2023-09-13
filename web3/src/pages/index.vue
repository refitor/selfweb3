<template>
    <div>
        <div v-show="!showSpin">
            <WebauthnPanel ref="webauthn" />
            <WalletPanel ref="walletPanel" :onAccountChanged="onAccountChanged" />
            <TOTPPanel v-if="showTOTP" ref="totpPanel" :getSelf="getSelf"/>
            <HomePanel v-show="showHomePanel && !showTOTP" ref="privatePanel" :getSelf="getSelf"/>
            <VaultPanel v-show="showPanels['SelfVault'] && !showTOTP" ref="SelfVault" :getSelf="getSelf"/>
        </div>
        <Spin size="large" fix v-if="showSpin"></Spin>
    </div>
</template>
<script>
import Web3 from "web3";
import TOTPPanel from './totp.vue';
import HomePanel from './home.vue';
import VaultPanel from './vault.vue';
import WalletPanel from './wallet.vue';
import WebauthnPanel from './webauthn.vue';
export default {
    components: {
        TOTPPanel,
        WalletPanel,
        HomePanel,
        VaultPanel,
        WebauthnPanel
    },
    inject: ["reload"],
    data() {
        return {
            selfID: '',
            connect: false,
            wasmPublic: '',
            web2Address: '',
            walletAddress: '',

            showTOTP: false,
            justVerify: false,
            showHomePanel: true,
            showPanels: {},

            panelName: '',
            backendPublicKey: '',
            afterVerifyFunc: null,

            apiPrefix: '',
            showSpin: false
        }
    },
    methods: {
        getSelf() {
            return this;
        },
        enableSpin(status) {
            this.showSpin = status;
        },
        webAuthnLogin() {
            let self = this;
            const go = new Go();
            this.enableSpin(true);
            WebAssembly.instantiateStreaming(fetch("selfweb3.wasm"), go.importObject)
            .then(function(result) {
                console.log('load wasm successed: ', result)
                go.run(result.instance);
                self.initBackend();
            })
        },
        onAccountChanged(action, network, address) {
            let self = this;
            if (action === 'connect') {
                this.connect = true;
                this.modelAuthID = address;
                this.walletAddress = address;
                this.webAuthnLogin();
            } else if (action === 'disconnect') {
                this.connect = false;
                this.walletAddress = '';
            } else {
                window.location.reload();
            }
        },
        initWeb3(selfID, web2Address) {
            let self = this;
            this.selfID = selfID;
            this.web2Address = web2Address;
            let message = 'SelfWeb3 Init: ' + (new Date()).getTime();
            self.signTypedData(message, function(sig) {
                var loadParams = [];
                loadParams.push(selfID);
                loadParams.push(sig);
                loadParams.push(Web3.utils.asciiToHex(message));
                self.$refs.walletPanel.Execute("call", "Load", self.walletAddress, 0, loadParams, function (loadResult) {
                    console.log('web3 contract: Web3Public successed: ', loadResult);
                    let recoverID = Web3.utils.hexToAscii(loadResult['recoverID']);
                    let web3Public = Web3.utils.hexToAscii(loadResult['web3Public']);
                    self.$refs.privatePanel.hasRegisted = true;
                    self.enableSpin(false);
                    self.$refs.privatePanel.init(recoverID, web3Public);
                }, function (err) {
                    self.enableSpin(false);
                    self.$Message.error('web3 contract: Web3Public failed');
                });
            })
        },
        initBackend() {
            let self = this;
            let response = {};
            WasmPublic(function(wasmResponse) {
                let queryMap = {};
                queryMap['kind'] = "web2Data";
                queryMap['params'] = "initWeb2";
                queryMap['userID'] = self.walletAddress;
                queryMap['public'] = JSON.parse(wasmResponse)['Data'];
                self.wasmPublic = JSON.parse(wasmResponse)['Data'];
                self.httpGet("/api/datas/load", queryMap, function(response) {
                    if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
                        self.$Message.error('load datas from web2 service failed: ', response.data['Error']);
                    } else {
                        let inputWeb2Key = "";
                        let web2Response = response.data['Data'];
                        WasmInit(self.walletAddress, inputWeb2Key, web2Response['Web2NetPublic'], web2Response['Web2Data'], function(initResponse) {
                            let wasmResp = {};
                            wasmResp['data'] = JSON.parse(initResponse);
                            if (wasmResp.data['Error'] !== '' && wasmResp.data['Error'] !== null && wasmResp.data['Error'] !== undefined) {
                                self.wasmCallback("Init", response.data['Error'], false);
                            } else {
                                console.log('backend init successed: ', wasmResp.data['Data']);
                                self.initWeb3(wasmResp.data['Data'], web2Response['Web2Address']);
                            }
                        });
                    }
                })
            });
        },
        getWalletAddress() {
            return this.walletAddress;
        },
        getWallet() {
            return this.$refs.walletPanel;
        },
        switchPanel(action, panelName, panelInitParam, afterVerifyFunc) {
            if (action === 'back' || action === '') {
                this.showPanels[panelName] = false;
                this.showHomePanel = !this.showHomePanel;
                // this.reload();
                return;
            }
            this.panelName = panelName;
            this.afterVerifyFunc = afterVerifyFunc;

            let self = this;
            this.showTOTP = true;
            this.$nextTick(function(){
                self.$refs.totpPanel.init(panelInitParam);
            });
        },
        afterVerify(hasVerified, panelInitParam, optionPanelName) {
            this.showTOTP = false;
            if (hasVerified === true) {
                console.log('verify successed: ', this.panelName, optionPanelName);
                if (this.afterVerifyFunc !== null && this.afterVerifyFunc !== undefined) {
                    this.afterVerifyFunc(panelInitParam);
                    return;
                }
                if (this.panelName === '' && optionPanelName !== undefined) this.panelName = optionPanelName;
                this.showHomePanel = !this.showHomePanel;
                this.showPanels[this.panelName] = true;
                this.$refs[this.panelName].init(panelInitParam);
            }
        },
        signTypedData(msg, callback) {
            var msgParams = [
                {
                    type: 'string',
                    name: 'Action',
                    value: msg
                }
            ]

            let self = this;
            let from = this.getWalletAddress();
            var params = [msgParams, from];
            var method = 'eth_signTypedData';
            this.$refs.walletPanel.getWeb3().currentProvider.sendAsync({
                method,
                params,
                from,
            }, function (error, result) {
                if (error || result.error) {
                    self.$Message.error('sign message failed at web3: ', msg, error);
                    console.log('sign message failed at web3: ', msg, error)
                    self.enableSpin(false);
                    return
                }
                if (callback !== null && callback !== undefined) callback(result.result);
            })
        },
        generatekey(num, needNO) {
            let library = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
            if (needNO === true) library = "0123456789";
            let key = "";
            for (var i = 0; i < num; i++) {
                let randomPoz = Math.floor(Math.random() * library.length);
                key += library.substring(randomPoz, randomPoz + 1);
            }
            return key;
        },
        wasmCallback(method, err, spinStatus) {
            if (spinStatus !== undefined) this.enableSpin(spinStatus);
            if (err === undefined || err === '') {
                this.$Message.success('exec wasm method successed: ' + method);
            } else {
                console.log('exec wasm method failed: ', method + ", ", err);
                this.$Message.error('exec wasm method failed: ' + method + ", " + err);
            }
        },
        httpGet(url, formdata, onResponse, onPanic) {
            this.$axios.get(this.apiPrefix + url, {params: formdata})
            .then(function(response) {
                if (onResponse !== undefined && onResponse !== null) onResponse(response);
            })
            .catch(function(e) {
                console.log(e);
            });
        },
        httpPost(url, formdata, onResponse, onPanic) {
            this.$axios.post(this.apiPrefix + url, formdata)
            .then(function(response) {
                if (onResponse !== undefined && onResponse !== null) onResponse(response);
            })
            .catch(function(e) {
                console.log(e);
            });
        }
    }
}
</script>