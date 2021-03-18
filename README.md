# Nelbert442 DERO Stargate Smart Contracts - Stargate RC2

## DeroDice.bas
Attempt at similar product as Ether Dice etc. Dice rolling game in which you can choose between a 2x and a 10x multiplier (increment by 1s [e.g. 2x, 3x, 4x, ... 10x]) and roll high or low.
The high and low numbers are defined as such:
```
    2x --> 50 or over --> 49 or under
    3x --> 67 or over --> 33 or under
    4x --> 75 or over --> 25 or under
    5x --> 80 or over --> 20 or under
    6x --> 84 or over --> 16 or under
    7x --> 86 or over --> 14 or under
    8x --> 88 or over --> 12 or under
    9x --> 89 or over --> 11 or under
    10x --> 90 or over --> 10 or under
```

There is a minimum wager/bet amount of 0.5 DERO and maximum wager/bet amount of 10 DERO, tuneable by TuneWagerParameters()

### Initialize Contract (initializes SC and makes you, the SIGNER(), the owner)

```
curl --request POST --data-binary @DeroDice.bas http://127.0.0.1:40403/install_sc
```

### e.x.1 (Roll High with 2x Multiplier - Wagering 2 DERO): 
```
curl http://127.0.0.1:40403/json_rpc -d '{"jsonrpc":"2.0","id":"0","method":"scinvoke","params":{"sc_dero_deposit":200000,"scid":"31b8f4ebb854ba2d5cd1ef0495f2d04dbd1e99b3754b03a8715c65028feef241","sc_rpc":[{"name":"entrypoint","datatype":"S","value":"RollDiceHigh"},{"name":"multiplier","datatype":"U","value":2}] }}' -H 'Content-Type: application/json'

https://testnetexplorer.dero.io/tx/cddf8f0c00a76179da2c61f314a063f420979fec749cd5d263f6e81b2fbc04c4
```

### e.x.2 (Roll Low with 2x Multiplier - Wagering 2 DERO):
```
curl http://127.0.0.1:40403/json_rpc -d '{"jsonrpc":"2.0","id":"0","method":"scinvoke","params":{"sc_dero_deposit":200000,"scid":"31b8f4ebb854ba2d5cd1ef0495f2d04dbd1e99b3754b03a8715c65028feef241","sc_rpc":[{"name":"entrypoint","datatype":"S","value":"RollDiceLow"},{"name":"multiplier","datatype":"U","value":2}] }}' -H 'Content-Type: application/json'

https://testnetexplorer.dero.io/tx/d7c2271b2aeaeec1e9e8734cb1c999721f7072e2cff197ae3a3bd8099d418186
```

### e.x.3 (TuneWagerParameters())
If you are the owner of the SC when initialized, you can then modify two of the built-in values: minWager, maxWager and sc_giveback. Once this function is ran, any transactions AFTER this has been ran will utilize these new values. This does not apply to previous transactions sent via the SC.

minWager: This is the value that users must use as a minimum bet, if they bet lower than this it will be rejected and returned to them.
maxWager: This is the value that users must us as a maximum bet, if they bet higher than this it will be rejected and returned to them.
sc_giveback: This is defining a percentage that the SC is giving to the Winnders. By default this value is set to 98%, however can be tuned with this function.

In this example, you can see that minWager is being set to 0.5 DERO (50000), the maxWager is being set to 10 DERO (1000000) and sc_giveback is being set to 98% (9800) given back to the Winner, keeping 2% for the SC.
```
curl http://127.0.0.1:40403/json_rpc -d '{"jsonrpc":"2.0","id":"0","method":"scinvoke","params":{"scid":"31b8f4ebb854ba2d5cd1ef0495f2d04dbd1e99b3754b03a8715c65028feef241","sc_rpc":[{"name":"entrypoint","datatype":"S","value":"TuneWagerParameters"},{"name":"minWager","datatype":"U","value":50000},{"name":"maxWager","datatype":"U","value":500000},{"name":"sc_giveback","datatype":"U","value":9500}] }}' -H 'Content-Type: application/json'
```

### e.x.4 (Donate to SC DERO Pool for Payouts - Donating 6 DERO):
```
curl http://127.0.0.1:40403/json_rpc -d '{"jsonrpc":"2.0","id":"0","method":"scinvoke","params":{"sc_dero_deposit":600000,"scid":"31b8f4ebb854ba2d5cd1ef0495f2d04dbd1e99b3754b03a8715c65028feef241","sc_rpc":[{"name":"entrypoint","datatype":"S","value":"Donate"}] }}' -H 'Content-Type: application/json'

https://testnetexplorer.dero.io/tx/8c4612646c4dc119a341fb828e037af98fc09f4bf7e2965faf5a03dcb6ca166a
```

### Future
* Make Private potentially, pending testing