#Instruction for Logx

In any module, since logx is already initialized, whenever
there is need for logging, just do:

`logx.info("message","key",value)
`


For example:

`int sampleTrash := 3
`
`moreTrash := "someone's code"
`
`logx.fail("somehow trash is generated", "sampleTrash", sampleTrash, "moreTrash", moreTrash)`