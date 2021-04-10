# Nelbert442 DERO Stargate Smart Contracts - Stargate RC2
For dARCH Event 0 - Round 2, I have setup [DeroDice](https://derodice.nelbert442.com) to host the FrontEnd stats related to leveraging this service. In order to get up and running, please use the 'Client Service' section / binary to get up and running locally and you can too start testing your odds!

For testnet purposes: I recommend generating new client / server wallets. Just to confirm no other services/dApps have used the ports defined (though not likely)

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

### Disclaimer
This implementation is under heavy development and this version is only for testing purposes at this time. Keep in mind that tx transfers can take an extended period of time depending on your client && server-side derod peers as well as current mining conditions on testnet making blocks longer or shorter in time. This iteration performs 3 separate TX prior to having a result, so with the previously mentioned details one can see some time to process.

I am fully intending to take this project even further and optimize/enhance the experience along the way as much as I can and ensure privacy never escapes the parties (client and server services) involved. In testing, flaws will be found and modifications will be made, the nature of the beast.

## DeroDice Service Utilization (dARCH Event 0 - Round 2 Addition)
DeroDice was originally created as a Smart Contract game in an attempt to mirror other similar dice games such as Ether Dice etc. With the advancements of DeroHE and the inclusion of [DERO Service capabilities](https://forum.dero.io/t/dero-service-model/1309), I decided to take this project a step further. This implementation includes both a client-side and server-side service. The server-side receiving TX details, generating a userKey which is then sent back to the client. Client-side takes in the received TX and details and interacts with the DeroDice.bas Smart Contract privately by using the intended wallet itself (hiding the source address using DeroDice) and only passing a randomly generated userkey which is then later used by the service for API/Stats capabilities. The userKey is only ever known by the client, server and SC where the only parties who can turn that into a DERO address are the client and service.

More documentation will be coming as I can make it to better explain the process, but it breaks down into 4 steps:

```
1) By using normal option 5 in CLI [transfer DERO], enter defined destination address (spit out by server service and visible on FE), enter the multiplier/function [2/RollDiceHigh i.e.], enter tx value to bet or use auto-bet [1]

2) Tx gets sent to the server service, which generates a userKey and sends details back in rpc.RPC_COMMENT such as: "2/RollDiceHigh/myUserKey"

3) Tx that was sent back to client gets picked up by client service, it reads rpc.RPC_COMMENT to receive: multiplier = 2, scFunction = RollDiceHigh, userKey = myUserKey

4) Client service sends details off to SC with params ingested
```

NOTE: This repo is still under development and will be a revolving project that I fully intend to take mainsteam come mainnet. The code nor developers are responsible for any DERO lost in the utilization of this application, please ensure to read fully and understand the layers involved (server and client)

![DERO Dice Home](assets/homePage.PNG?raw=true)
![Custom Bet Address](assets/customBetAddress.PNG?raw=true)
![Set Bet Address](assets/setBetAddress.PNG?raw=true)

### Server Service (owner of userKey gen, API, FrontEnd)
DeroDice-Server.go contains the server service code, also see releases for the pre-built binary for ease-of use (though I always recommend building binaries yourself). This can be ran locally, or on a server intended for hosting the API/Frontend publicly for user consumption.

NOTE: Ensure that the rpc-server (wallet) port used is different between the server service and client service if ran on the same machine.

![DERO Dice Server Service](assets/serverService.PNG?raw=true)

DeroDice-Server usage and help output below (heavy development and will be modified in future iterations)
```
DeroDice-Server
DERO Dice Service (server): A 2 way service implementation for interacting with a Smart Contract while ensuring input variables are as private as possible

Usage:
  DeroDice-Server [options]
  DeroDice-Server -h | --help

Options:
  -h --help     Show this screen.
  --daemon-address=<host:port>	Use daemon instance at <host>:<port> or https://domain
  --rpc-server-address=<127.0.0.1:40403>	connect to service (server) wallet
  --scid=<73535cbe3254d0943d52e4b2b94dcf98d29868b364c21046c57bb8575218474f>	define SCID that is leveraged (NOTE: MUST BE SAME ON CLIENT AND SERVER)
  --api-address=<0.0.0.0:8224>	API (non-SSL) will be enabled at the defined address (or defaulted to 0.0.0.0:8224)
  --ssl-api-address=<0.0.0.0:8225>	if defined, API (SSL) will be enabled at the defined address. apifullchain.cer && apicert.key in the same dir is required
  --frontend-port=<8080>	if defined, frontend (non-SSL) will be enabled
  --ssl-frontend-port=<8181>	if defined, frontend (SSL) will be enabled. fefullchain.cer && fecert.key in the same dir is required
```

### Client Service (local TX processing and SC sending)
DeroDice-Client.go contains the client service code, also see releases for the pre-built binary for ease-of use (though I always recommend building binaries yourself). This is intended to be ran locally alongside your normal cli-wallet (or other rpc-server capable wallet(s)). 

NOTE: Ensure that the rpc-server (wallet) port used is different between the server service and client service if ran on the same machine.

![DERO Dice Client Service](assets/clientService.PNG?raw=true)

DeroDice-Client usage and help output below (heavy development and will be modified in future iterations)
```
DeroDice-Client
DERO Dice Service (client): A 2 way service implementation for interacting with a Smart Contract while ensuring input variables are as private as possible

Usage:
  DeroDice-Client [options]
  DeroDice-Client -h | --help

Options:
  -h --help     Show this screen.
  --rpc-server-address=<127.0.0.1:40403>	connect to service (client) wallet
  --scid=<5984617fb00799d0184eef6cefa3750cb9f812058378a0bbb70a62264a76347f>	define SCID that is leveraged (NOTE: MUST BE SAME ON CLIENT AND SERVER)
```

### Backend DB for Server/Client Service (graviton)
In usual form with DERO projects I have taken on recently, I leverage [Graviton](https://github.com/deroproject/graviton) for the backend DB store.

## DeroDice Direct SC Utilization (dARCH Event 0 - Round 1 Implementation)

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