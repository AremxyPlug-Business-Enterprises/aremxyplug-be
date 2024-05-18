package airtime

import "github.com/aremxyplug-be/db/models/telcom"

func (a *AirtimeConn) SaveRecipient(data telcom.AirtimeRecipient) error {
	if err := a.db.SaveAirtimeRecipient(data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) GetRecipients(username string) ([]telcom.AirtimeRecipient, error) {
	resp, err := a.db.GetAirtimeRecipients(username)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) UpdateRecipient(userID string, data telcom.AirtimeRecipient) error {
	if err := a.db.EditAirtimeRecipient(userID, data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) DeleteRecipient(name, userID string) error {
	if err := a.db.DeleteAIrtimeRecipient(name, userID); err != nil {
		return err
	}

	return nil
}
