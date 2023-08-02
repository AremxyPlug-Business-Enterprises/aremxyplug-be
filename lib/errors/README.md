# Remitng Terrors

## Description
Remitng error core library, Remit Terror implements go builtin error interface.<br>
All Remitng go applications must use this library to handle errors.<br><br>
See more: https://fcmbuk.atlassian.net/wiki/spaces/ROAV/pages/387645441/Errors.


## Usage
- Initialize a simple Terror
```golang
import (
    "github.com/remitng/zebra/errors"
)

error := NewTerror(
    code, //int
    errorType, //string
    message, //string
    detail, //string
)`

// Output when error.Error() executed
{
   "error":{
      "code":7001,
      "type":"InvalidPhoneNumberException",
      "message":"Provided phone number is already attached to Roava account",
      "detail":"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue"
   }
}
```

- Initialize a Terror with additional attributes
```golang
import (
    "github.com/remitng/zebra/errors"
)

error := NewTerror(
    code, //int
    errorType, //string
    message, //string
    detail, //string
    WithStatus(http.StatusOK) //...TerrorOptionalAttrs,
)

// Output when error.Error() executed
{
   "error":{
      "code":7001,
      "type":"InvalidPhoneNumberException",
      "message":"Provided phone number is already attached to Roava account",
      "status":200,
      "detail":"This phone number is already attached to a Roava account. Kindly recheck the phone number or logon to continue"
   }
}
```
>NOTE: You can pass multiple Terror additional attributes

### Available additional attributes
- `WithStatus(status int)`
- `WithInstance(instance string)`
- `WithTraceID(traceID string)`
- `WithHelp(help string)`