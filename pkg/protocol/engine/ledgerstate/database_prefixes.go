package ledgerstate

const (
	StoreKeyPrefixLedgerSlotIndex byte = 0

	// StoreKeyPrefixOutput defines the prefix for Output and Spent storage.
	StoreKeyPrefixOutput byte = 1

	// StoreKeyPrefixOutputSpent defines the prefix for Track spent/unspent Outputs.
	StoreKeyPrefixOutputSpent   byte = 2
	StoreKeyPrefixOutputUnspent byte = 3

	// StoreKeyPrefixSlotDiffs defines the prefix for Slot diffs.
	StoreKeyPrefixSlotDiffs byte = 4

	StoreKeyPrefixStateTree byte = 5
)

/*
   LedgerState Database

   SlotIndex:
   ===============
   Key:
       StoreKeyPrefixLedgerSlotIndex
                1 byte

   Value:
       iotago.SlotIndex
          8 bytes

   Output:
   =======
   Key:
       StoreKeyPrefixOutput + iotago.OutputID
             1 byte         +     34 bytes

   Value:
       BlockID   +   SlotIndex  + TransactionCreationTimestamp (time.Time) + iotago.Output.Serialized()
       40 bytes  +    8 bytes   +                  8 byte      s           +   1 byte type + X bytes

   Spent Output:
   ================
   Key:
       StoreKeyPrefixSpent + iotago.OutputID
            1 byte         +     34 bytes

   Value:
       TargetTransactionID (iotago.TransactionID) + TransactionAcceptedSlotIndex (iotago.SlotIndex) + TransactionCreationTimestamp (time.Time)
                  32 bytes                        +                    8 bytes                      +       8 bytes

   Unspent Output:
   ===============
   Key:
       StoreKeyPrefixUnspent + iotago.OutputID
             1 byte          +     34 bytes

   Value:
       Empty


   Slot diffs:
   ================
   Key:
       StoreKeyPrefixSlotDiffs + iotago.SlotIndex
                 1 byte        +     8 bytes

   Value:
       OutputCount  +  OutputCount  *  iotago.OutputID   + SpentCount +  SpentCount *    iotago.OutputID
         4 bytes    +  (OutputCount *    34 bytes)       +   4 bytes  + (SpentCount *       34 bytes)

*/
