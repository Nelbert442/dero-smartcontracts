/*  DeroDice.bas
    Dice rolling game in which you can choose between a 2x and a 10x multiplier (increment by 1s [e.g. 2x, 3x, 4x, ... 10x]) and roll high or low.
    The high and low numbers are defined as such:

    2x --> 50 or over --> 49 or under
    3x --> 67 or over --> 33 or under
    4x --> 75 or over --> 25 or under
    5x --> 80 or over --> 20 or under
    6x --> 84 or over --> 16 or under
    7x --> 86 or over --> 14 or under
    8x --> 88 or over --> 12 or under
    9x --> 89 or over --> 11 or under
    10x --> 90 or over --> 10 or under

    There is a minimum wager/bet amount of 0.5 DERO and maximum wager/bet amount of 10 DERO

    Author: Nelbert442
    Media Handles: @Nelbert442
*/

Function InitializePrivate() Uint64
    10 STORE("owner", SIGNER())
    20 STORE("minWager", 50000) // Minimum wager set to 0.5 DERO
    30 STORE("maxWager", 1000000) // Maximum wager set to 10 DERO
    40 STORE("sc_giveback", 9800)   // SC will give reward 98% of deposits/winnings, 2.0 % is accumulated for owner to withdraw as well as SC to keep for processing fees etc.
    50 STORE("balance", 0)

    60 STORE("Over-x2", 50)
    61 STORE("Under-x2", 49)
    65 STORE("Over-x3", 67)
    66 STORE("Under-x3", 33)
    70 STORE("Over-x4", 75)
    71 STORE("Under-x4", 25)
    75 STORE("Over-x5", 80)
    76 STORE("Under-x5", 20)
    80 STORE("Over-x6", 84)
    81 STORE("Under-x6", 16)
    85 STORE("Over-x7", 86)
    86 STORE("Under-x7", 14)
    90 STORE("Over-x8", 88)
    91 STORE("Under-x8", 12)
    95 STORE("Over-x9", 89)
    96 STORE("Under-x9", 11)
    100 STORE("Over-x10", 90)
    101 STORE("Under-x10", 10)

    190 STORE("minMultiplier", 2) // TODO: Add to TuneWagerParameters, if and only if develop loop to generate the Over-x# and Under-x# values. Can do with math and looping for var creation incrementing tempcounter, just need to circle back to it
    191 STORE("maxMultiplier", 10) // TODO: Add to TuneWagerParameters, if and only if develop loop to generate the Over-x# and Under-x# values. Can do with math and looping for var creation incrementing tempcounter, just need to circle back to it

    210 RETURN 0
End Function

Function TuneWagerParameters(minWager Uint64, maxWager Uint64, sc_giveback Uint64) Uint64
    10 IF LOAD("owner") == SIGNER() THEN GOTO 30 // Validate owner is one calling this function, otherwise return
	25 RETURN 1

    30 IF minWager <= 0 THEN GOTO 40
    35 STORE("minWager", minWager)
    40 IF maxWager <= 0 THEN GOTO 50
    45 STORE("maxWager", maxWager)
    50 IF sc_giveback <= 0 THEN GOTO 60
    55 STORE("sc_giveback", sc_giveback)

    60 RETURN 0
End Function

Function error(errorMessage String, userKey String) Uint64
    10 DIM return_balance as Uint64
    11 DIM txid as String
    12 LET txid = TXID()
    15 LET return_balance = DEROVALUE()
    20 IF (LOAD("balance") + return_balance) > return_balance THEN GOTO 105 // seems silly, but loads balance and if balance = 0 then + return_balance is never going to be > return_balance, so take 1% off for fee
    30 LET return_balance = 9900 * return_balance / 10000 // no need to store this fee for withdraw, it's safe to have a small amount stored in SC that is not withdrawable for tx fees later "cost of doing business"
    40 STORE("balance", DEROVALUE() - return_balance) // should only be ran when no funds (new SC initialization etc.)

    105 SEND_DERO_TO_ADDRESS(SIGNER(), return_balance)
    110 IF userKey == "" THEN GOTO 999 ELSE GOTO 150
    150 STORE(userKey, errorMessage)

    999 RETURN 0
End Function

Function Donate() Uint64
	10 DIM balance, tempcounter as Uint64
    11 DIM errorMessage,txid as String
    15 IF DEROVALUE() == 0 THEN GOTO 85  // if value is 0, simply return

	50 LET balance = LOAD("balance") + DEROVALUE()
	60 STORE("balance", balance)
    65 LET txid = TXID()

	85 RETURN 0
End Function

// destinationAddress and userKey are not required for functionality to work. A normal user can curl with "" values for destinationAddress and userKey. These params are for service utilization
Function RollDiceHigh(multiplier Uint64, userKey String) Uint64
    10 DIM rolledNum, targetNumber, payoutAmount, minWager, maxWager, minMultiplier, maxMultiplier, currentTopoHeight as Uint64
    11 DIM errorMessage,txid,sendToAddr,userMessage as String
    12 LET txid = TXID()
    13 LET currentTopoHeight = BLOCK_TOPOHEIGHT()

    // If playing without service model (or not defined destinationAddress)
    30 IF EXISTS(SIGNER()) == 1 THEN GOTO 31 ELSE GOTO 32
    31 IF LOAD(SIGNER()) == currentTopoHeight THEN GOTO 600 // Check for SIGNER() variable value and see if it is equal to current topoheight, if so then error out and return value, otherwise store topoheight and continue
    32 STORE(SIGNER(), currentTopoHeight)
    33 LET sendToAddr = SIGNER()
    34 GOTO 40

    40 LET minWager = LOAD("minWager")
    41 LET maxWager = LOAD("maxWager")
    42 LET minMultiplier = LOAD("minMultiplier")
    43 LET maxMultiplier = LOAD("maxMultiplier")
    45 IF DEROVALUE() < minWager THEN GOTO 800 // If value is less than stored minimum wager (e.g. 0.5), Error and send DERO back
    50 IF DEROVALUE() > maxWager THEN GOTO 800 // If value is greater than stored maximum wager (e.g. 10), Error and send DERO back
    55 LET payoutAmount = LOAD("sc_giveback") * DEROVALUE() * multiplier / 10000
    
    // IF exists "Over-x" + multiplier, then proceed. Else exit because this means they did not supply a multiplier within 2 - 10.
    60 IF EXISTS("Over-x" + multiplier) == 1 THEN GOTO 70 ELSE GOTO 900

    70 LET rolledNum = RANDOM(99) // Randomly choose number between 0 and 99
    80 LET targetNumber = LOAD("Over-x" + multiplier)
    90 IF rolledNum >= targetNumber THEN GOTO 100 ELSE GOTO 500

    100 IF LOAD("balance") < payoutAmount THEN GOTO 700 // If balance cannot cover the potential winnings, error out and send DERO back to SIGNER() [keep some % if balance is 0]
    105 IF userKey == "" THEN GOTO 120
    107 LET userMessage = "Win:" + multiplier + "," + currentTopoHeight + "," + txid
    110 STORE(userKey, userMessage)
    120 SEND_DERO_TO_ADDRESS(sendToAddr, payoutAmount)
    125 STORE("balance", LOAD("balance") + (DEROVALUE() - payoutAmount))
    130 RETURN 0

    // PRINTF "Thanks for playing, however unfortunately you rolled a %d which is lower than %d. TRY AGAIN!" rolledNum targetNumber
    500 IF userKey == "" THEN GOTO 504
    502 LET userMessage = "Lose:" + multiplier + "," + currentTopoHeight + "," + txid
    503 STORE(userKey, userMessage)
    504 STORE("balance", LOAD("balance") + DEROVALUE())
    505 RETURN 0

    600 RETURN error("Err: DeroDice already played at current topoheight, please wait a few seconds then try again", sendToAddr, userKey)

    700 RETURN error("Err: Not enough funds available in DeroDice. Please try again later or submit a ticket for funds to be added to pool", sendToAddr, userKey)

    800 LET errorMessage = "Err: Incorrect Wager amount. Please use between " + minWager + " and " + maxWager + " DERO"
    820 RETURN error(errorMessage, sendToAddr, userKey)

    900 LET errorMessage = "Err: Incorrect multiplier. Please use between " + minMultiplier + " and " + maxMultiplier
    920 RETURN error(errorMessage, sendToAddr, userKey)
End Function

// destinationAddress and userKey are not required for functionality to work. A normal user can curl with "" values for destinationAddress and userKey. These params are for service utilization
Function RollDiceLow(multiplier Uint64, userKey String) Uint64
    10 DIM rolledNum, targetNumber, payoutAmount, minWager, maxWager, minMultiplier, maxMultiplier, currentTopoHeight as Uint64
    11 DIM errorMessage,txid,sendToAddr,userMessage as String
    12 LET txid = TXID()
    13 LET currentTopoHeight = BLOCK_TOPOHEIGHT()

    // If playing without service model (or not defined destinationAddress)
    30 IF EXISTS(SIGNER()) == 1 THEN GOTO 31 ELSE GOTO 32
    31 IF LOAD(SIGNER()) == currentTopoHeight THEN GOTO 600 // Check for SIGNER() variable value and see if it is equal to current topoheight, if so then error out and return value, otherwise store topoheight and continue
    32 STORE(SIGNER(), currentTopoHeight)
    33 LET sendToAddr = SIGNER()
    34 GOTO 40

    40 LET minWager = LOAD("minWager")
    41 LET maxWager = LOAD("maxWager")
    42 LET minMultiplier = LOAD("minMultiplier")
    43 LET maxMultiplier = LOAD("maxMultiplier")
    45 IF DEROVALUE() < minWager THEN GOTO 800 // If value is less than stored minimum wager (e.g. 0.5), Error and send DERO back
    50 IF DEROVALUE() > maxWager THEN GOTO 800 // If value is greater than stored maximum wager (e.g. 10), Error and send DERO back
    55 LET payoutAmount = LOAD("sc_giveback") * DEROVALUE() * multiplier / 10000
    
    // IF exists "Under-x" + multiplier, then proceed. Else exit because this means they did not supply a multiplier within 2 - 10.
    60 IF EXISTS("Under-x" + multiplier) == 1 THEN GOTO 70 ELSE GOTO 900

    70 LET rolledNum = RANDOM(99) // Randomly choose number between 0 and 99
    80 LET targetNumber = LOAD("Under-x" + multiplier)
    90 IF rolledNum <= targetNumber THEN GOTO 100 ELSE GOTO 500

    100 IF LOAD("balance") < payoutAmount THEN GOTO 700 // If balance cannot cover the potential winnings, error out and send DERO back to SIGNER() [keep some % if balance is 0]
    105 IF userKey == "" THEN GOTO 120
    107 LET userMessage = "Win:" + multiplier + "," + currentTopoHeight + "," + txid
    110 STORE(userKey, userMessage)
    120 SEND_DERO_TO_ADDRESS(sendToAddr, payoutAmount)
    125 STORE("balance", LOAD("balance") + (DEROVALUE() - payoutAmount))
    130 RETURN 0

    // PRINTF "Thanks for playing, however unfortunately you rolled a %d which is lower than %d. TRY AGAIN!" rolledNum targetNumber
    500 IF userKey == "" THEN GOTO 504
    502 LET userMessage = "Lose:" + multiplier + "," + currentTopoHeight + "," + txid
    503 STORE(userKey, userMessage)
    504 STORE("balance", LOAD("balance") + DEROVALUE())
    505 RETURN 0

    600 RETURN error("Err: DeroDice already played at current topoheight, please wait a few seconds then try again", sendToAddr, userKey)

    700 RETURN error("Err: Not enough funds available in DeroDice. Please try again later or submit a ticket for funds to be added to pool", sendToAddr, userKey)

    800 LET errorMessage = "Err: Incorrect Wager amount. Please use between " + minWager + " and " + maxWager + " DERO"
    820 RETURN error(errorMessage, sendToAddr, userKey)

    900 LET errorMessage = "Err: Incorrect multiplier. Please use between " + minMultiplier + " and " + maxMultiplier
    920 RETURN error(errorMessage, sendToAddr, userKey)
End Function

// This function is used to change owner 
// owner is an string form of address 
Function TransferOwnership(newowner String) Uint64 
    10  IF LOAD("owner") == SIGNER() THEN GOTO 30 
    20  RETURN 1
    30  STORE("tmpowner",ADDRESS_RAW(newowner))
    40  RETURN 0
End Function

// Until the new owner claims ownership, existing owner remains owner
Function ClaimOwnership() Uint64 
    10  IF LOAD("tmpowner") == SIGNER() THEN GOTO 30 
    20  RETURN 1
    30  STORE("owner",SIGNER()) // ownership claim successful
    40  RETURN 0
End Function

// if signer is owner, withdraw any requested funds
// if everthing is okay, they will be showing in signers wallet
Function Withdraw(amount Uint64) Uint64 
    10  IF LOAD("owner") == SIGNER() THEN GOTO 30 
    20  RETURN 1
    30  SEND_DERO_TO_ADDRESS(SIGNER(),amount)
    40  RETURN 0
End Function