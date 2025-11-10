package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
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

func (t *SmartContract) DepositChecking(ctx contractapi.TransactionContextInterface, customId string, amount int) error {
	account, err := loadAccount(ctx, customId)
	if err != nil {
		return fmt.Errorf("Account %s not found", customId)
	}
	account.CheckingBalance += amount
	err = saveAccount(ctx, account)
	if err != nil { // PandaTODO: 对于这样的分支，现实中如何判断是否要成为路径条件呢？
		return fmt.Errorf("Put state failed in DepositChecking")
	}

	// PandaCond: account exists
	return nil
}
func (t *SmartContract) WriteCheck(ctx contractapi.TransactionContextInterface, customId string, amount int) error {
	account, err := loadAccount(ctx, customId)
	if err != nil {
		return fmt.Errorf("Account %s not found", customId)
	}
	if account.CheckingBalance < amount {
		return fmt.Errorf("Insufficient funds in source savings account")
	}
	account.CheckingBalance -= amount
	err = saveAccount(ctx, account)
	if err != nil {
		return fmt.Errorf("Put state failed in WriteCheck")
	}
	// PandaCond: account exists && account.CheckingBalance >= amount
	return nil
}

func (t *SmartContract) TransactSavings(ctx contractapi.TransactionContextInterface, customId string, amount int) error {
	account, err := loadAccount(ctx, customId)
	if err != nil {
		return fmt.Errorf("Account %s not found", customId)
	}
	// since the contract is only used for perfomance testing, we ignore this check
	//if amount < 0 && account.SavingsBalance < (-amount) {
	//	return errormsg("Insufficient funds in source savings account")
	//}
	account.SavingsBalance += amount
	err = saveAccount(ctx, account)
	if err != nil {
		return fmt.Errorf("Put state failed in TransactionSavings")
	}
	// PandaCond: account exists
	return nil
}

// var path_state = 0

// 2 reads and 2 writes
func (t *SmartContract) SendPayment(ctx contractapi.TransactionContextInterface, src_customId string, dst_customId string, amount int) error {
	destAccount, err1 := loadAccount(ctx, dst_customId)
	sourceAccount, err2 := loadAccount(ctx, src_customId)
	if err1 != nil || err2 != nil {
		// path_state += 1
		return fmt.Errorf("Account [ %s or %s ] not found", src_customId, dst_customId)
	}
	// path_state += 2

	// if sourceAccount.CheckingBalance < amount {
	// 	return errormsg("Insufficient funds in source checking account")
	// }
	sourceAccount.CheckingBalance -= amount
	destAccount.CheckingBalance += amount
	err1 = saveAccount(ctx, sourceAccount)
	err2 = saveAccount(ctx, destAccount)
	// time.Sleep(50 * time.Millisecond)
	if err1 != nil || err2 != nil {
		// path_state += 4
		return fmt.Errorf("Putstate failed in sendPayment")
	}
	// path_state += 8

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

func (t *SmartContract) Amalgamate(ctx contractapi.TransactionContextInterface, dst_customId string, src_customId string) error {
	destAccount, err1 := loadAccount(ctx, dst_customId)
	sourceAccount, err2 := loadAccount(ctx, src_customId)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("Account [ %s or %s ] not found", src_customId, dst_customId)
	}
	err1 = saveAccount(ctx, sourceAccount)
	err2 = saveAccount(ctx, destAccount)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("Put state failed in sendPayment")
	}

	return nil
}

func (t *SmartContract) Query(ctx contractapi.TransactionContextInterface, customId string) (*Account, error) {
	key := accountKey(customId)
	accountBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to read from world state.")
	}
	if accountBytes == nil {
		return nil, fmt.Errorf("Account %s does not exist", customId)
	}
	account := new(Account)
	_ = json.Unmarshal(accountBytes, account)
	// fmt.Printf("%+v\n", account)
	return account, nil
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
