/**
 * Copyright 2017 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * ------------------------------------------------------------------------------
 */

package tests

import (
  "time"
  "testing"
  "sawtooth_burrow_evm/client"
  . "sawtooth_burrow_evm/common"
	. "sawtooth_burrow_evm/protobuf/evm_pb2"
  ptypes "burrow/permission/types"
  sdk "sawtooth_sdk/client"
)

const (
  PRIV      = "5J7bEeWs14sKkz7yVHfVc2FXKfBe6Hb5oNZxxTKqKZCgjbDTuUj"
  PRIV2     = "5JWiDuaXRhkJQomz3GNsBVYhvcY4NysrWNKwXsWY31hU7BrvShN"
  INIT      = "6060604052341561000c57fe5b5b6101c48061001c6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631ab06ee51461005c578063812600df146100855780639507d39a146100a5578063c20efb90146100d9575bfe5b341561006457fe5b61008360048080359060200190919080359060200190919050506100f9565b005b341561008d57fe5b6100a36004808035906020019091905050610116565b005b34156100ad57fe5b6100c36004808035906020019091905050610148565b6040518082815260200191505060405180910390f35b34156100e157fe5b6100f76004808035906020019091905050610166565b005b8060006000848152602001908152602001600020819055505b5050565b600160006000838152602001908152602001600020540160006000838152602001908152602001600020819055505b50565b6000600060008381526020019081526020016000205490505b919050565b600160006000838152602001908152602001600020540360006000838152602001908152602001600020819055505b505600a165627a7a723058203d60fc69e0e52544039deda2b37c9f9ab67fde5fda29b4dea3088495ff7c096c0029"
  SET_0_42  = "1ab06ee50000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002a"
  DEC_0     = "c20efb900000000000000000000000000000000000000000000000000000000000000000"
  WAIT      = 300
)

func TestPermissions(t *testing.T) {
  client := client.New("http://rest-api:8080")
  priv := sdk.WifToPriv(PRIV)
  priv2 := sdk.WifToPriv(PRIV2)
  init := sdk.MustDecode(INIT)
  nonce := uint64(0)

  // At the beginning of the chain, all actions are allowed because the global
  // permissions have not been set.

  // Create an External Account so we can change permissions. Since global
  // permissions have not been set yet, the account is created with all
  // permissions.
  _, err := client.CreateExternalAccount(priv, nil, nil, nonce)
  if err != nil {
    t.Error(err.Error())
  }
  nonce += 1
  time.Sleep(WAIT * time.Millisecond)

  // Now that we have an account, we can disable all global permissions
  globalPermsAddr := GlobalPermissionsAddress().Bytes()
  zeroPerms := &EvmPermissions{Perms: uint64(0), SetBit: uint64(ptypes.AllPermFlags)}
  err = client.SetPermissions(priv, globalPermsAddr, zeroPerms, nonce)
  if err != nil {
    t.Error(err.Error())
  }
  nonce += 1
  time.Sleep(WAIT * time.Millisecond)

  // Check that global permissions were set
  globalPermsEntry, err := client.Get(globalPermsAddr)
  if err != nil {
    t.Error(err.Error())
  }
  setPerms := globalPermsEntry.GetAccount().GetPermissions()
  if setPerms.Perms != uint64(0) || setPerms.SetBit != uint64(ptypes.AllPermFlags) {
    t.Errorf(
      "Global permissions not set correctly. Should be (0, ALL) but got (%v, %v)",
      setPerms.Perms, setPerms.SetBit,
    )
  }

  // Test that new accounts cannot be created without authorization
  addr2, err := client.CreateExternalAccount(priv2, nil, nil, 0)
  if err != nil {
    t.Error(err.Error())
  }
  time.Sleep(WAIT * time.Millisecond)
  entry, err := client.Get(addr2)
  if err != nil {
    t.Error(err.Error())
  }
  if entry != nil {
    t.Error("Account created but creation is disabled.")
  }

  // Create a new account that can create and call contracts
  perms2 := &EvmPermissions{
    Perms: uint64(ptypes.CreateContract | ptypes.Call),
    SetBit: uint64(ptypes.CreateContract | ptypes.Call),
  }
  addr2, err = client.CreateExternalAccount(priv2, priv, perms2, nonce)
  if err != nil {
    t.Error(err.Error())
  }
  nonce2 := uint64(1)
  nonce += 1
  time.Sleep(WAIT * time.Millisecond)
  entry, err = client.Get(addr2)
  if err != nil {
    t.Error(err.Error())
  }
  if entry == nil {
    t.Error("Failed to create new account.")
  }

  // Verify the account can't change permissions
  err = client.SetPermissions(priv2, globalPermsAddr, &EvmPermissions{
    Perms: uint64(ptypes.AllPermFlags),
    SetBit: uint64(ptypes.AllPermFlags),
  }, nonce2)
  if err != nil {
    t.Error(err.Error())
  }
  globalPermsEntry, err = client.Get(globalPermsAddr)
  if err != nil {
    t.Error(err.Error())
  }
  setPerms = globalPermsEntry.GetAccount().GetPermissions()
  if setPerms.Perms != uint64(0) || setPerms.SetBit != uint64(ptypes.AllPermFlags) {
    t.Errorf("Permissions changed but changing permissions is disabled.")
  }

  // Verify the account can deploy a contract
  contractAddr, err := client.CreateContractAccount(priv2, init, nil, nonce2, 1000)
  if err != nil {
   t.Error(err.Error())
  }
  nonce2 += 1
  time.Sleep(WAIT * time.Millisecond)
  contractEntry, err := client.Get(contractAddr)
  if err != nil {
    t.Error(err.Error())
  }
  if contractEntry == nil {
    t.Error("Failed to deploy contract.")
  }

  // Verify the account can call a contract
  _, err = client.MessageCall(priv2, contractAddr, sdk.MustDecode(SET_0_42), nonce2, 1000)
  if err != nil {
    t.Error(err.Error())
  }
  nonce2 += 1
  time.Sleep(WAIT * time.Millisecond)
  contractEntry, err = client.Get(contractAddr)
  if err != nil {
    t.Error(err.Error())
  }

  key := "ad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5"
  val := "000000000000000000000000000000000000000000000000000000000000002a"
  pair := contractEntry.GetStorage()[0]
  if sdk.MustEncode(pair.GetKey()) != key || sdk.MustEncode(pair.GetValue()) != val {
    t.Error("Failed to call contract.")
  }

  // Disable the accounts permissions
  err = client.SetPermissions(priv, addr2, zeroPerms, nonce)
  if err != nil {
    t.Error(err.Error())
  }
  nonce += 1
  time.Sleep(WAIT * time.Millisecond)

  // Verify the account can't deploy a contract
  contractAddr2, err := client.CreateContractAccount(priv2, init, nil, nonce2, 1000)
  if err != nil {
   t.Error(err.Error())
  }
  nonce2 += 1
  time.Sleep(WAIT * time.Millisecond)
  contractEntry2, err := client.Get(contractAddr2)
  if err != nil {
    t.Error(err.Error())
  }
  if contractEntry2 != nil {
    t.Error("Deployed contract but contract deployment is disabled for this account.")
  }

  // Verify the account can't call a contract
  _, err = client.MessageCall(priv2, contractAddr, sdk.MustDecode(SET_0_42), nonce2, 1000)
  if err != nil {
    t.Error(err.Error())
  }
  time.Sleep(WAIT * time.Millisecond)
  contractEntry, err = client.Get(contractAddr)
  if err != nil {
    t.Error(err.Error())
  }

  pair = contractEntry.GetStorage()[0]
  if sdk.MustEncode(pair.GetValue()) != val {
    t.Error("Contract called but calling is disabled for this account.")
  }
}
