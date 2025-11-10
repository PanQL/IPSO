package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PanQL/fabric-contract-api-go/contractapi"
	peer "github.com/PanQL/fabric-protos-go/peer"
)

var namespace = hexdigest("smallbank")[:6]

// SmartContract provides functions for managing a Account
type SmartContract struct {
	contractapi.Contract
}

type Account struct {
	Type            string `json:"Type"`
	CustomId        string `json:"customId"`
	CustomName      string `json:"customName"`
	SavingsBalance  int    `json:"savingsBalance"`
	CheckingBalance int    `json:"checkingBalance"`
}

func hexdigest(str string) string {
	hash := sha512.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	return strings.ToLower(hex.EncodeToString(hashBytes))
}

func accountKey(id string) string {
	return namespace + hexdigest(id)[:64]
}

func saveAccount(ctx contractapi.TransactionContextInterface, account *Account) error {
	accountBytes, err := json.Marshal(account)
	if err != nil {
		return err
	}
	key := accountKey(account.CustomId)
	return ctx.GetStub().PutState(key, accountBytes)
}
func loadAccount(ctx contractapi.TransactionContextInterface, id string) (*Account, error) {
	key := accountKey(id)
	accountBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	res := Account{}
	err = json.Unmarshal(accountBytes, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	fmt.Println("nothing init")
	return nil
}

func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, customId string, customName string, savingsBalance int, checkingBalance int) error {
	key := accountKey(customId)
	data, err := ctx.GetStub().GetState(key)
	if data != nil {
		return fmt.Errorf("Can not create duplicated account")
	}

	// checking, errcheck := strconv.Atoi(checkingBalance)
	// if errcheck != nil {
	// 	return fmt.Errorf(" create_account, checking balance should be integer")
	// }
	// saving, errsaving := strconv.Atoi(savingsBalance)
	// if errsaving != nil {
	// 	return fmt.Errorf(" create_account, saving balance should be integer")
	// }
	account := &Account{
		Type:            "account",
		CustomId:        customId,
		CustomName:      customName,
		SavingsBalance:  savingsBalance,
		CheckingBalance: checkingBalance,
	}

	err = saveAccount(ctx, account)
	if err != nil {
		return fmt.Errorf("Put state failed when create account %s", customId)
	}
	return nil

}

// 2 reads and 2 writes
func (t *SmartContract) SendPayment(ctx contractapi.TransactionContextInterface, src_customId string, dst_customId string, amount int) error {
	destAccount, err1 := loadAccount(ctx, dst_customId)
	sourceAccount, err2 := loadAccount(ctx, src_customId)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("Account [ %s or %s ] not found", src_customId, dst_customId)
	}

	if sourceAccount.CheckingBalance < amount {
		return nil // give up the transfer
		// return fmt.Errorf("Insufficient funds in source checking account")
	}
	sourceAccount.CheckingBalance -= amount
	destAccount.CheckingBalance += amount
	err1 = saveAccount(ctx, sourceAccount)
	err2 = saveAccount(ctx, destAccount)
	// time.Sleep(50 * time.Millisecond)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("Putstate failed in sendPayment")
	}

	// PandaCond: sourceAccount exists && destAccount exists && sourceAccount.CheckingBalance >= amount
	ctx.GetStub().HintHotPathCond(accountKey(src_customId), peer.HotPathCondType_CHECKING_LARGER_EQUAL, amount)
	return nil
}

// 1 read and 2 writes
func (t *SmartContract) SendPaymentCRDT(ctx contractapi.TransactionContextInterface, src_customId string, dst_customId string, amount int) error {
	sourceAccount, err2 := loadAccount(ctx, src_customId)
	if err2 != nil {
		return fmt.Errorf("Account [ %s or %s ] not found", src_customId, dst_customId)
	}

	// if sourceAccount.CheckingBalance < amount {
	// 	return errormsg("Insufficient funds in source checking account")
	// }
	sourceAccount.CheckingBalance -= amount
	err1 := saveAccount(ctx, sourceAccount)
	account := &Account{
		Type:            "crdt",
		CustomId:        dst_customId,
		CustomName:      dst_customId,
		SavingsBalance:  0,
		CheckingBalance: amount,
	}
	err2 = saveAccount(ctx, account)
	// time.Sleep(50 * time.Millisecond)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("Putstate failed in sendPayment")
	}

	return nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))

	if err != nil {
		fmt.Printf("Error create fabcar chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting fabcar chaincode: %s", err.Error())
	}
}
