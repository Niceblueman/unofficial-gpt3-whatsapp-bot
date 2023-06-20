# API Documentation

## Introduction
Welcome to the API documentation! This API allows you to send messages to phone numbers. Below, you'll find details on how to use the API endpoints and examples in cURL format.

## Send Message
Sends a message to the provided phone numbers.

- Endpoint: `/send-message`
- Method: `POST`
### Request Headers
The following header is required for authentication:

- `X-API-Key`: Your API key for authentication.

### Request Body
The request body should be a JSON object with the following properties:

```json
{
  "numbers": ["+1234567890", "+9876543210"],
  "message": "test bot: Hello, World!"
}
```
- numbers: An array of phone numbers to send the message to.
- message: The message content (maximum 250 characters).

```shell
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "numbers": ["+1234567890", "+9876543210"],
    "message": "Hello, World!"
  }' \
  https://whatsapp.dup.company/send-message
```
### Error Responses
In case of errors, the API will respond with appropriate status codes and error messages. Here are some possible error scenarios:

 - If the request body is invalid or missing required fields:
    - Status Code: 400 Bad Request
    - Response Body: Invalid request body or specific error message
- If the message length exceeds the maximum limit of 250 characters:
    - Status Code: 400 Bad Request
    - Response Body: Message exceeds the maximum length of 250 characters
- If an invalid phone number is provided:
    - Status Code: 400 Bad Request
    - Response Body: Invalid phone number: {phone number}

### Conclusion
That's it! You now have all the necessary information to start using the API. If you have any further questions or issues, feel free to reach out to our support team [![Telegram Logo](https://upload.wikimedia.org/wikipedia/commons/thumb/8/82/Telegram_logo.svg/23px-Telegram_logo.svg.png)](https://t.me/Capbarbas).

Happy messaging!