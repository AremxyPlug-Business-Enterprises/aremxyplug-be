package airtime

import "github.com/aremxyplug-be/db/models/telcom"

func (a *AirtimeConn) SaveRecipient(userID string, data telcom.Recipient) error {
	if err := a.db.SaveTelcomRecipient(userID, data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) GetRecipients(username string) (telcom.TelcomRecipient, error) {
	resp, err := a.db.GetTelcomRecipients(username)
	if err != nil {
		return telcom.TelcomRecipient{}, err
	}

	return resp, nil
}

func (a *AirtimeConn) UpdateRecipient(userID string, data telcom.Recipient) error {
	if err := a.db.EditTelcomRecipient(userID, data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) DeleteRecipient(recipientID int, userID string) error {
	if err := a.db.DeleteTelcomRecipient(recipientID, userID); err != nil {
		return err
	}

	return nil
}
