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
                        <Button :disabled ="!hasRegisted" @click="beforeRecover()" type="primary" style="margin: 10px;">Recover</Button>
                        <Button :disabled ="!hasRegisted" @click="executeAction('SelfVault')" type="primary" style="margin: 10px;">SelfVault</Button>
                        <!-- <Button :disabled ="!hasRegisted || web3Key === ''" @click="logout()" type="primary" style="margin: 10px;">Logout</Button> -->
                        <!-- <Button :disabled ="!hasRegisted || web3Key === ''" @click="executeAction('cryptoPanel', '')" type="primary" style="margin: 10px;">Encrypt-Decrypt</Button> -->
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
                <Button v-if="modalMode === 'recover'" type="primary" @click="recover()" style="margin-right: 10px;">Confirm</Button>
                <Button v-if="modalMode === 'verify'" type="primary" @click="afterRecover()" style="margin-right: 10px;">Confirm</Button>
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
                            let needView = true;//self.items.data[params.index]['key'] !== 'Wallet' && self.items.data[params.index]['key'] !== 'Contract';
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

            modelKey: '',
            modelValue: '',
            qrcodeUrl: '',
            qrcodeSize: 200,

            web3Key: '',
            recoverID: '',
            web3Public: '',
            hasRegisted: false
        }
    },
    mounted: function () {
    },
    methods: {
        openLink(url) {
            window.open(url, '_blank');
        },
        init(recoverID, web3Key, web3Public) {
            this.web3Key = web3Key;
            this.recoverID = recoverID;
            this.web3Public = web3Public;
            const contractAddr = this.$parent.getSelf().getWallet().contractAddrMap[this.$parent.getSelf().getWallet().networkId];
            this.addKV('Wallet', {'btnName': 'View', 'value': this.$parent.getSelf().getWalletAddress(), 'url': 'https://etherscan.io/token/' + this.$parent.getSelf().getWalletAddress()}, true);
            this.addKV('Contract', {'btnName': 'View', 'value': contractAddr, 'url': 'https://etherscan.io/token/' + contractAddr}, false);
            // this.addKV('Encrypt-Decrypt', {'value': this.$parent.getSelf().generatekey(16, false), 'btnName': 'Test'}, false);
            // this.addKV('SelfVault', {'value': contractAddr, 'btnName': 'Launch'}, false);
            console.log('init HomePanel: ', recoverID, web3Key, web3Public);
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

            // wasm
            let response = {};
            let userID = self.$parent.getSelf().getWalletAddress();
            WasmRegister(userID, this.modelKey, function(wasmResponse) {
                response['data'] = JSON.parse(wasmResponse);
                if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
                    self.$parent.getSelf().wasmCallback("Register", response.data['Error']);
                } else {
                    self.$parent.getSelf().wasmCallback("Register");
                    var registParams = [];
                    var web3Key = response.data['Data']['Web3Key'];
                    var recoverID = response.data['Data']['RecoverID'];
                    var web3Public = response.data['Data']['Web3Public'];
                    registParams.push(Web3.utils.asciiToHex(recoverID));
                    registParams.push(Web3.utils.asciiToHex(web3Key));
                    registParams.push(Web3.utils.asciiToHex(web3Public));
                    self.$parent.getSelf().enableSpin(true);
                    self.$parent.getSelf().getWallet().Execute("send", "Register", self.$parent.getSelf().getWalletAddress(), 0, registParams, function (result) {
                        self.$parent.getSelf().enableSpin(false);
                        let recoverID = self.modelKey;
                        self.resetModal();
                        self.hasRegisted = true;

                        // store web2Private
                        let formdata = new FormData();
                        formdata.append("userID", userID);
                        formdata.append("recoverID", recoverID);
                        formdata.append("kind", 'selfweb3.web2Data');
                        formdata.append("params", response.data['Data']['Web2Data']);
                        self.$parent.getSelf().httpPost("/api/datas/store", formdata, function(storeResponse) {
                            if (storeResponse.data['Error'] == '') {
                                self.showQRcode(response.data['Data']['QRCode']);
                                setTimeout(function() {
                                    self.qrcodeUrl = '';
                                    window.location.reload();
                                }, 60000);
                                // Please add an account through Google Authenticator within 1 minutes
                            }
                        })
                    }, function (err) {
                        self.$Message.error('selfWeb3 register at contract failed');
                        self.$parent.getSelf().enableSpin(false);
                    })
                }
            })
        },
        beforeRecover() {
            let self = this;
            self.resetModal();
            self.modalMode = 'recover';
            self.popModal = true;
        },
        recover() {
            let self = this;
            if (this.modelKey === '') {
                this.$Message.error('pushID must be non-empty');
                return;
            }
            this.popModal = false;

            let response = {};
            let userID = self.$parent.getSelf().getWalletAddress();
            WasmAuthorizeCode(userID, this.modelKey, function(wasmResponse) {
                response['data'] = JSON.parse(wasmResponse);
                if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
                    self.$parent.getSelf().wasmCallback("Register", response.data['Error']);
                } else {
                    // store web2Private
                    let formdata = new FormData();
                    formdata.append("userID", userID);
                    formdata.append("kind", 'email');
                    formdata.append("params", response.data['Data']);
                    formdata.append("public", self.$parent.getSelf().wasmPublic);
                    self.$parent.getSelf().httpPost("/api/datas/forward", formdata, function(forwardResponse) {
                        if (forwardResponse.data['Error'] == '') {
                            self.$parent.getSelf().enableSpin(false);
                            self.$Message.success('email push successed for recovery');
                            self.resetModal();
                            self.modalMode = 'verify';
                            self.popModal = true;
                        }
                    })
                }
            })
        },
        afterRecover() {
            let self = this;
            if (this.modelKey === '') {
                this.$Message.error('encrypted privateKey must be non-empty');
                return;
            }
            this.popModal = false; 
            this.$parent.getSelf().enableSpin(true);

            let response = {};
            let recoverMap = {"method": "ResetTOTPKey", "recoverID": self.recoverID, "web3Key": self.web3Key, "web3Public": self.web3Public};
            let userID = self.$parent.getSelf().getWalletAddress();
            WasmVerify(this.$parent.getSelf().getWalletAddress(), this.modelKey, 'email', this.action, JSON.stringify(recoverMap), function(wasmResponse) {
                response['data'] = JSON.parse(wasmResponse);
                console.log('ResetTOTPKey: ', response)
                if (response.data['Error'] !== '' && response.data['Error'] !== null && response.data['Error'] !== undefined) {
                    self.$parent.getSelf().wasmCallback("WasmVerify", response.data['Error'], false);
                } else {
                    self.$parent.getSelf().enableSpin(false);
                    // store web2Private
                    let formdata = new FormData();
                    formdata.append("userID", userID);
                    formdata.append("kind", 'selfweb3.web2Data');
                    formdata.append("params", response.data['Data']['Web2Data']);
                    self.$parent.getSelf().httpPost("/api/datas/store", formdata, function(storeResponse) {
                        if (storeResponse.data['Error'] == '') {
                            self.showQRcode(response.data['Data']['QRCode']);
                            setTimeout(function() {
                                self.qrcodeUrl = '';
                                window.location.reload();
                            }, 60000);
                            // Please add an account through Google Authenticator within 1 minutes
                        }
                    })
                }
            })
        },
        executeAction(name, params) {
            let self = this;
            const contractAddr = this.$parent.getSelf().getWallet().contractAddrMap[this.$parent.getSelf().getWallet().networkId];
            if (name === 'SelfVault') {
                let web3Map = {"method": "SelfVault", "web3Key": self.web3Key, "web3Public": self.web3Public};
                this.$parent.getSelf().switchPanel('dapp', name, JSON.stringify(web3Map));
            } else {
                this.$parent.getSelf().switchPanel(name, name, contractAddr);
            }
        },
        showQRcode(totpKey) {
            // Google authenticator doesn't like equal signs
            var walletAddress = this.$parent.getSelf().getWalletAddress();
            let walletAddr = walletAddress.substring(0, 4) + "..." + walletAddress.substring(walletAddress.length - 4, walletAddress.length);

            // to create a URI for a qr code (change totp to hotp if using hotp)
            const totpName = 'selfWeb3-' + this.$parent.getSelf().getWallet().networkId + ':' + walletAddr;
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