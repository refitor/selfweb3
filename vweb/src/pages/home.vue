<style scoped>
.layout {
    /* border: 1px solid #d7dde4; */
    position: relative;
    border-radius: 4px;
    overflow: hidden;
    width: 100%;
    text-align: center;
}

.layout-content-center {
    display: inline-block;

    margin-top: 3%;

    max-width: 60%;
}
</style>
<template>
    <div>
        <div class="layout">
            <div class="layout-content-center">
                <div>
                    <div style="margin: 10px;">
                        <h2 style="margin-bottom: 3px;">SelfWeb3</h2>
                        <Button :disabled="hasRegisted" @click="modalMode = 'register'; popModal = true" type="primary" style="margin: 10px;">Regist</Button>
                        <!-- <Button :disabled ="!hasRegisted" @click="beforeRecover()" type="primary" style="margin: 10px;">Recover</Button> -->
                        <Select v-model="resetKind" style="width:90px; text-align: center;" placeholder="Reset" @on-select="beforeRecover">
                            <Option value="TOTP">TOTP</Option>
                            <Option disabled value="Wallet">Wallet</Option>
                            <Option disabled value="Web2Key">Web2Key</Option>
                        </Select>
                        <Button :disabled ="!hasRegisted" @click="executeAction('SelfVault')" type="primary" style="margin: 10px;">SelfVault</Button>
                        <Table border style="margin-top: 8px;" no-data-text="empty key/value list" :columns="items.columns" :data="items.data"></Table>
                    </div>
                    <VueQrcode v-if="qrcodeUrl !== ''" :value="qrcodeUrl" :options="{ width: 150 }" />
                    <h3 v-if="qrcodeUrl !== ''" style="text-align: center;">Please add an account via Google Authenticator within 1 minute and refresh the page.</h3>
                </div>
            </div>
        </div>
        <Modal
            v-model="popModal"
            :footer-hide="hideFooter"
            class-name="vertical-center-modal">
            <p style="text-align: center;margin-bottom: 10px;">{{placeHolderMap[modalMode].title}}</p>
            <Input v-model="modelKey" type="email" :placeholder="placeHolderMap[modalMode].placeholder"><span slot="prepend">{{placeHolderMap[modalMode].name}}</span></Input>
            <!--Input v-if="modalMode === 'verify'" v-model="modelValue" type="text"><span slot="prepend" style="margin-top: 10px;">Value</span></Input-->
            <div style="text-align: center; margin-top: 15px;">
                <Button v-if="modalMode === 'register'" type="primary" @click="register()" style="margin-right: 10px;">Confirm</Button>
                <Button v-if="modalMode === 'recover'" type="primary" @click="emailSend()" style="margin-right: 10px;">Confirm</Button>
                <Button v-if="modalMode === 'verify'" type="primary" @click="emailVerify()" style="margin-right: 10px;">Confirm</Button>
                <Button @click="popModal = false; modalReadonly = false;">Cancel</Button>
            </div>
        </Modal>
    </div>
</template>
<script>
import Web3 from "web3";
import CryptoJS from 'crypto-js'
import emailjs from '@emailjs/browser';
import VueQrcode from '@chenfengyuan/vue-qrcode';
import * as selfweb3 from '../logic/index.js';
export default {
    components: {
        VueQrcode
    },
    inject: ["reload"],
    data() {
        return {
            hideFooter: true,
            popModal: false,
            modalMode: 'register',
            placeHolderMap: {
                'verify': {'name': 'Code', 'placeholder': 'Enter dynamic code...', 'title': 'recovery verify'},
                'register': {'name': 'Email', 'placeholder': 'Enter recover email...', 'title': 'user registration'},
                'recover': {'name': 'Email', 'placeholder': 'Enter recover email......', 'title': 'social recovery'}
            },

            items:{
                columns:[
                    {
                        title: 'Key',
                        key: 'key',
                        align: 'center',
                        minWidth:50
                    },
                    {
                        title: 'Value',
                        key: 'value',
                        align: 'center',
                        minWidth:200
                    },
                    {
                        title: 'Action',
                        key: 'action',
                        width: 200,
                        align: 'center',
                        render: (h, params) => {
                            let btns = []
                            let self = this;
                            let needView = self.items.data[params.index]['data']['btnName'] !== undefined;
                            btns.push(h('Button', {
                                    props: {
                                        type: 'primary',
                                        disabled: needView ? false : true
                                    },
                                    on: {
                                        click: () => {
                                            console.log(self.items.data[params.index])
                                            if (self.items.data[params.index]['data']['url'] === undefined) self.executeAction(self.items.data[params.index]['key'], self.items.data[params.index]['data']['value']);
                                            else self.openLink(self.items.data[params.index]['data']['url']);
                                        }
                                    }
                                }, needView ? self.items.data[params.index]['data']['btnName']:'---')
                            );
                            return h('div', btns);
                        }
                    }
                ],
                data:[]
            },
            resetKind: '',

            modelKey: '',
            modelValue: '',
            qrcodeUrl: '',
            qrcodeSize: 200,

            web3Key: '',
            recoverID: '',
            web3Public: '',
            hasRegisted: true
        }
    },
    mounted: function () {
    },
    methods: {
        openLink(url) {
            window.open(url, '_blank');
        },
        init(recoverID, web3Public) {
            this.recoverID = recoverID;
            this.web3Public = web3Public;
            const web2Address = this.$parent.getSelf().web2Address;
            this.addKV('SelfID', {'value': this.$parent.getSelf().selfAddress}, true);
            const contractAddr = this.$parent.getSelf().getWallet().contractAddrMap[this.$parent.getSelf().getWallet().networkId];
            this.addKV('Wallet', {'btnName': 'View', 'value': this.$parent.getSelf().getWalletAddress(), 'url': 'https://etherscan.io/token/' + this.$parent.getSelf().getWalletAddress()}, false);
            this.addKV('Contract', {'btnName': 'View', 'value': contractAddr, 'url': 'https://etherscan.io/token/' + contractAddr}, false);
            this.addKV('Web2Address', {'btnName': 'View', 'value': web2Address, 'url': 'https://etherscan.io/token/' + web2Address}, false);
            // this.addKV('Encrypt-Decrypt', {'value': this.$parent.getSelf().generatekey(16, false), 'btnName': 'Test'}, false);
            // this.addKV('SelfVault', {'value': contractAddr, 'btnName': 'Launch'}, false);
        },
        addKV(k, v, bReset) {
            if (bReset === true) this.items.data = [];
            if (k === 'Encrypt-Decrypt') {
                this.items.data.pop(k);
            }
            this.items.data.push({'key': k, 'value': v['value'], 'data': v});
        },
        resetModal() {
            this.modelKey = '';
            this.modelValue = '';
        },
        register() {
            var self = this;
            if (this.modelKey === '') {
                this.$Message.error('encryption name must be non-empty');
                return;
            }
            this.popModal = false;

            let selfAddress = self.$parent.getSelf().selfAddress;;
            let walletAddress = self.$parent.getSelf().getWalletAddress();
            selfweb3.GetUser().Register(walletAddress, selfAddress, self.modelKey, function(address, qrcode){
                self.showQRcode(address, qrcode);
                setTimeout(function() {
                    self.qrcodeUrl = '';
                    window.location.reload();
                }, 60000);
            })

            //// no logic js
            // self.$parent.getSelf().enableSpin(true);

            // // wasm
            // let response = {};
            // let userID = self.$parent.getSelf().getWalletAddress();
            // WasmRegister(userID, this.modelKey, function(wasmResponse) {
            //     response['data'] = JSON.parse(wasmResponse);
            //     if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
            //         self.$parent.getSelf().wasmCallback("Register", response.data['Error'], false);
            //     } else {
            //         self.$parent.getSelf().wasmCallback("Register");
            //         var registParams = [];
            //         registParams.push(self.$parent.getSelf().selfAddress);
            //         registParams.push(Web3.utils.asciiToHex(response.data['Data']['RecoverID']));
            //         registParams.push(Web3.utils.asciiToHex(response.data['Data']['Web3Key']));
            //         registParams.push(Web3.utils.asciiToHex(response.data['Data']['Web3Public']));
            //         // 流程: contract.Register ===> webAuthnRegister ===> /api/datas/store ===> TOTP QRCode
            //         self.$parent.getSelf().getWallet().Execute("send", "Register", self.$parent.getSelf().getWalletAddress(), 0, registParams, function (result) {
            //             let recoverID = self.modelKey;
            //             self.hasRegisted = true;
            //             self.resetModal();

            //             self.$parent.getSelf().$refs.webauthn.webRegister(userID, function(){
            //                 self.$parent.getSelf().enableSpin(false);
            //                 self.storeWeb2Data(userID, recoverID, response.data['Data']['Web2Data'], response.data['Data']['QRCode']);
            //             }, function() {
            //                 self.$parent.getSelf().enableSpin(false);
            //                 self.$Message.error('webAuthn register failed');
            //             });
            //         }, function (err) {
            //             self.$parent.getSelf().enableSpin(false);
            //             self.$Message.error('web3 contract: register failed');
            //         })
            //     }
            // })
        },
        beforeRecover() {
            let self = this;
            self.resetModal();
            self.modalMode = 'recover';
            self.popModal = true;
        },
        emailSend() {
            let self = this;
            if (this.modelKey === '') {
                this.$Message.error('pushID must be non-empty');
                return;
            }
            this.popModal = false;
            this.$parent.getSelf().enableSpin(true);

            let selfAddress = self.$parent.getSelf().selfAddress;;
            let walletAddress = self.$parent.getSelf().getWalletAddress();
            selfweb3.GetVerify().BeginEmailVerify(self.$parent.getSelf().getWalletAddress(), this.modelKey, function(){
                self.$Message.success('email push successed for recovery');
                self.$parent.getSelf().enableSpin(false);
                self.resetModal();
                self.modalMode = 'verify';
                self.popModal = true;
            })

            //// no logic js
            // let response = {};
            // let userID = self.$parent.getSelf().getWalletAddress();
            // WasmAuthorizeCode(userID, this.modelKey, function(wasmResponse) {
            //     response['data'] = JSON.parse(wasmResponse);
            //     if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
            //         self.$parent.getSelf().wasmCallback("Register", response.data['Error'], false);
            //     } else {
            //         // store web2Private
            //         let formdata = new FormData();
            //         formdata.append("userID", userID);
            //         formdata.append("kind", 'email');
            //         formdata.append("params", response.data['Data']);
            //         formdata.append("public", self.$parent.getSelf().wasmPublic);
            //         self.$parent.getSelf().httpPost("/api/datas/forward", formdata, function(forwardResponse) {
            //             if (forwardResponse.data['Error'] == '') {
            //                 self.$Message.success('email push successed for recovery');
            //                 self.$parent.getSelf().enableSpin(false);
            //                 self.resetModal();
            //                 self.modalMode = 'verify';
            //                 self.popModal = true;
            //             } else {
            //                 self.$parent.getSelf().enableSpin(false);
            //             }
            //         })
            //     }
            // })
        },
        emailVerify() {
            let self = this;
            if (this.modelKey === '') {
                this.$Message.error('encrypted privateKey must be non-empty');
                return;
            }
            this.popModal = false;

            let selfAddress = self.$parent.getSelf().selfAddress;;
            let walletAddress = self.$parent.getSelf().getWalletAddress();
            selfweb3.GetUser().Reset(walletAddress, selfAddress, this.modelKey, self.resetKind, function(resetParams){
                if (self.resetKind === 'TOTP') {
                    self.showQRcode(selfAddress, resetParams);
                    setTimeout(function() {
                        self.qrcodeUrl = '';
                        window.location.reload();
                    }, 60000);
                }
            })

            //// no logic js
            // let selfAddress = self.$parent.getSelf().selfAddress;
            // let userID = self.$parent.getSelf().getWalletAddress();
            // if (self.resetKind === "TOTP") {
            //     let resetMap = {"method": "ResetTOTPKey", "recoverID": self.recoverID, "web3Public": self.web3Public};
            //     self.verifyEmail(this.modelKey, resetMap, function(wasmEmailResponse) {
            //         self.relationVerify(false, true, function(){
            //             self.storeWeb2Data(userID, '', wasmEmailResponse['Web2Data'], wasmEmailResponse['QRCode']);
            //         })
            //     })
            // } else if (self.resetKind === "Wallet") {
            //     console.log(self.resetKind)
            // } else if (self.resetKind === "Web2Key") {
            //     console.log(self.resetKind)
            // }
        },
        executeAction(name, params) {
            let self = this;
            const contractAddr = this.$parent.getSelf().getWallet().contractAddrMap[this.$parent.getSelf().getWallet().networkId];
            if (name === 'SelfVault') {
                //// no logic js
                // // TOTP校验成功后获取到WebAuthnKey, 触发webAuthnLogin
                // // 流程: WebAuthnKey获取 ===> webAuthnLogin ===> switchPanel
                // self.relationVerify(true, true, function() {
                //     self.$parent.getSelf().afterVerifyFunc = null;
                //     self.$parent.getSelf().afterVerify(true, '', name)
                // }, function() {
                //     self.$Message.error('Can not init SelfVault with relationVerify failed');
                // })

                // self.$parent.getSelf().afterVerifyFunc = null;
                // self.$parent.getSelf().afterVerify(true, '', name);

                self.$parent.getSelf().RunTOTP(name, function(code) {
                    let selfAddress = self.$parent.getSelf().selfAddress;;
                    let walletAddress = self.$parent.getSelf().getWalletAddress();
                    selfweb3.GetUser().EnterDapp(walletAddress, selfAddress, code, function() {
                        self.$parent.getSelf().afterVerifyFunc = null;
                        self.$parent.getSelf().afterVerify(true, '', name);
                    })
                })
            } else {
                this.$parent.getSelf().switchPanel(name, name, contractAddr);
            }
        },
        storeWeb2Data(userID, recoverID, web2Data, qrcode) {
            // store web2Private
            let self = this;
            let formdata = new FormData();
            formdata.append("userID", userID);
            formdata.append("kind", 'web2Data');
            formdata.append("params", web2Data);
            formdata.append("recoverID", recoverID);
            self.$parent.getSelf().httpPost("/api/datas/store", formdata, function(storeResponse) {
                if (storeResponse.data['Error'] == '') {
                    self.showQRcode(qrcode);
                    setTimeout(function() {
                        self.qrcodeUrl = '';
                        window.location.reload();
                    }, 60000);
                    // Please add an account through Google Authenticator within 1 minutes
                } else {
                    self.$Message.error('store web2Data failed: ' + storeResponse.data['Error']);
                }
            })
        },
        verifyEmail(code, emailMap, callback) {
            let self = this;
            let response = {};
            let userID = self.$parent.getSelf().getWalletAddress();
            WasmVerify(userID, code, 'email', JSON.stringify(emailMap), function(wasmResponse) {
                let response = JSON.parse(wasmResponse);
                console.log('verifyEmail: ', response)
                if (response['Error'] !== '' && response['Error'] !== null && response['Error'] !== undefined) {
                    self.$parent.getSelf().wasmCallback("WasmVerify", response['Error'], false);
                } else {
                    if (callback !== undefined && callback !== null) callback(response['Data']);
                }
            })
        },
        // 关联验证流程: web3合约: 钱包签名校验(提取web3Public) ---> TOTP校验 ---> web3合约: zk证明校验, 钱包签名校验(提取web3Key) ---> webAuthn登录校验
        relationVerify(bTOTP, bWebAuthn, callback, failed) {
            let self = this;
            let userID = self.$parent.getSelf().getWalletAddress();
            let verifyWebAuthn = function(params) {
                // 关联性webAuthn校验，依赖zkParams链上校验提取web3Key
                let loadParams = [];
                loadParams.push(self.$parent.getSelf().selfAddress);
                loadParams = loadParams.concat(params);
                self.$parent.getSelf().$refs.walletPanel.Execute("call", "Web3Key", userID, 0, loadParams, function (loadResult) {
                    console.log('web3 contract: Load from contract successed: ', loadResult);
                    let web3Map = {"method": "WebAuthnKey", "web3Key": Web3.utils.hexToAscii(loadResult), "web3Public": self.web3Public};
                    WasmHandle(userID, JSON.stringify(web3Map), function(wasmWebAuthnResponse) {
                        self.$parent.getSelf().$refs.webauthn.webLogin(self.$parent.getSelf().getWalletAddress(), JSON.parse(wasmWebAuthnResponse)['Data'], function() {
                            if (callback !== undefined && callback !== null) callback();
                        }, function() {
                            if (failed !== undefined && failed !== null) failed();
                        })
                    })
                }, function (err) {
                    self.$Message.error('web3 contract: Load from contract failed');
                    if (failed !== undefined && failed !== null) failed();
                })
            }

            if (bTOTP === true) {
                let relateTimes = "1";
                let web3Map = {"method": "RelationVerify", "web3Key": '', "web3Public": self.web3Public, "action": "dapp"};
                self.$parent.getSelf().switchPanel('RelationVerify', '', JSON.stringify(web3Map), function(wasmTOTPResponse){
                    if (bWebAuthn === true) {
                        self.packRelateVerifyParams(web3Map['action'], wasmTOTPResponse, verifyWebAuthn);
                    } else {
                        if (callback !== undefined && callback !== null) callback(wasmTOTPResponse);
                    }
                })
            } else if (bWebAuthn === true) {
                verifyWebAuthn([]);
            }
        },
        // relateKind: Email, TOTP, WebAuthn
        packRelateVerifyParams(action, verifyParams, callback) {
            console.log('packRelateVerifyParams: ', action, verifyParams);
            let self = this;
            let queryMap = {};
            queryMap['action'] = action;
            queryMap['kind'] = 'relateVerify';
            queryMap['nonce'] = verifyParams['nonce'];
            queryMap['userID'] = self.$parent.getSelf().getWalletAddress();
            self.$parent.getSelf().httpGet("/api/datas/load", queryMap, function(response) {
                if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
                    self.$Message.error('load datas from web2 service failed: ', response.data['Error']);
                } else {
                    const relateVerifyParams = response.data['Data'];
                    console.log('before packRelateVerifyParams: ', relateVerifyParams);
                    const merkleParams = self.$parent.getSelf().$refs.walletPanel.getWeb3().eth.abi.encodeParameter(
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
        },
        showQRcode(selfAddress, totpKey) {
            // Google authenticator doesn't like equal signs
            let userID = selfAddress;
            // let walletAddress = this.$parent.getSelf().getWalletAddress();
            let selfID = userID.substring(0, 4) + "..." + userID.substring(userID.length - 4, userID.length);

            // to create a URI for a qr code (change totp to hotp if using hotp)
            const totpName = 'selfWeb3-' + this.$parent.getSelf().getWallet().networkId + ':' + selfID;
            this.qrcodeUrl = 'otpauth://totp/' + totpName + '?secret=' + totpKey.replace(/=/g,'');
        },
        pageWidth(){
            var winWidth=0;
            if (window.innerWidth){
                winWidth = window.innerWidth;
            }
            else if ((document.body) && (document.body.clientWidth)){
                winWidth = document.body.clientWidth;
            }
            if (document.documentElement && document.documentElement.clientWidth){
                winWidth = document.documentElement.clientWidth;
            }
            return winWidth;
        },
        pageHeight(){
            var winHeight=0;
            if (window.innerHeight){
                winHeight = window.innerHeight;
            }
            else if ((document.body) && (document.body.clientHeight)){
                winHeight = document.body.clientHeight;
            }
            if (document.documentElement && document.documentElement.clientHeight){
                winHeight = document.documentElement.clientHeight;
            }
            return winHeight;
        }
    }
}
</script>