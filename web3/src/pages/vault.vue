<style scoped>
.layout{
    /* border: 1px solid #d7dde4; */
    position: relative;
    border-radius: 4px;
    overflow: hidden;
    width: 100%;
    text-align: center;
}
.layout-content-center{
    display: inline-block;
    
    margin-top: 10%;

    max-width: 450px;
}
</style>
<template>
    <div>
        <div class="layout">
            <div class="layout-content-center">
                <h2 id="authTitle" style="text-align: center; margin-bottom: 20px;">SelfVault</h2>
                <!-- <h3 style="text-align: center; margin-bottom: 20px;"><a href="javascript:void(0)" @click="openLink('https://github.com/refitor/selfweb3/blob/main/contracts/selfVault.sol')">{{contractAddress}}</a></h3> -->
                <Input v-model="modelWalletBalance" type="text" readonly ><span slot="prepend">Wallet</span></Input>
                <Input v-model="modelBalance" type="text" readonly style="margin-top: 20px;"><span slot="prepend">Balance</span></Input>
                <Row style="margin-top: 20px;">
                    <Col span="21">
                        <Input v-model="modelAmount" type="text"><span slot="prepend">Amount</span></Input>
                    </Col>
                    <Col span="3" style="margin-top: 5px;">
                        <a href="javascript:void(0)" @click="modelAmount = modelWalletBalance" style="margin-left: 5px;">Max</a>
                    </Col>
                </Row>
                <Button @click="back()" type="primary" style="margin-top: 20px;">Back</Button>
                <Button :disabled="parseInt(modelAmount) <= 0" @click="deposit()" type="primary" style="margin-top: 20px; margin-left: 10px;">Deposit</Button>
                <Button :disabled="parseInt(modelAmount) <= 0" @click="withdraw()" type="primary" style="margin-top: 20px; margin-left: 10px;">Withdraw</Button>
                <Button v-show="triggerTx !== ''" @click="openLink(triggerTx)" type="primary" style="margin-top: 20px; margin-left: 10px;">Transaction</Button>
            </div>
        </div>
    </div>
</template>
<script>
import * as selfweb3 from '../logic/index.js';
export default {
    inject: ["reload"],
    data() {
        return {
            walletAddress: '',
            verifyCount: 0,
            triggerTx: '',
            contractAddress: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2',

            modelBalance: 0,
            modelWalletBalance: 0,
            modelAmount: 0,
        }
    },
    mounted: function() {
    },
    methods: {
        openLink(url) {
            window.open(url, '_blank');
        },
        back() {
            this.$parent.getSelf().switchPanel('back', 'SelfVault');
        },
        init(web3Key) {
            let self = this;
            this.balance();
            console.log('init SelfVault: ', web3Key);
        },
        balance() {
            let self = this;
            let selfAddress = self.$parent.getSelf().selfAddress;
            let walletAddress = self.$parent.getSelf().getWalletAddress();
            selfweb3.GetVault().Balance(walletAddress, selfAddress, function(balance){
                self.modelBalance = balance;
            })
        },
        deposit() {
            let self = this;
            let amount = parseInt(this.modelAmount);
            if (amount <= 0) {
                this.$Message.error('amount must be valid');
                return
            }

            self.$parent.getSelf().RunTOTP("SelfVault", function(code) {
                let selfAddress = self.$parent.getSelf().selfAddress;;
                let walletAddress = self.$parent.getSelf().getWalletAddress();
                selfweb3.GetVault().Deposit(walletAddress, selfAddress, self.modelAmount, code, function() {
                    self.balance();
                })
            })
        },
        withdraw() {
            let self = this;
            let amount = parseInt(this.modelAmount);
            if (amount <= 0) {
                this.$Message.error('amount must be valid');
                return
            }

            self.$parent.getSelf().RunTOTP("SelfVault", function(code) {
                let selfAddress = self.$parent.getSelf().selfAddress;;
                let walletAddress = self.$parent.getSelf().getWalletAddress();
                selfweb3.GetVault().Withdraw(walletAddress, selfAddress, self.modelAmount, code, function() {
                    self.balance();
                })
            })
        }
    }
}
</script>