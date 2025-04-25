## Setup and Running

### Prerequisites

* Go (version 1.18 or later recommended)

### Instructions

1.  **Clone the repository:**
    

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Run the application:**
    ```bash
    go run cmd/main.go
    ```
    The server will start on `http://localhost:8080` by default.

## How to Test [cite: 6]

You can use tools like `curl`, Postman, or Insomnia to interact with the API endpoints.


All endpoints are prefixed with `/api/v1`.

### Health Check

* **GET /health**
    * Description: Checks the health of the API service.
    * Response (Success 200):
        ```json
        {
            "status": "ok"
        }
        ```
    * Example:
        ```bash
        curl http://localhost:8080/health
        ```

### User Management

* **POST /users**
    * Description: Creates a new user with a specified ID and a default initial balance (e.g., 1000.0).
    * Request Body:
        ```json
        {
            "user_id": "string"
        }
        ```
    * Response (Success 201): User object (including ID, balance, created_at, updated_at).
    * Response (Error 400): Validation error (e.g., missing `user_id`).
    * Response (Error 409): User with the given ID already exists.
    * Example:
        ```bash
        curl -X POST http://localhost:8080/api/v1/users \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "charlie789"
        }'
        ```

* **GET /users**
    * Description: Retrieves a list of all registered users.
    * Response (Success 200): Array of user objects.
    * Example:
        ```bash
        curl http://localhost:8080/api/v1/users
        ```

* **GET /users/{userId}**
    * Description: Retrieves details for a specific user by their ID.
    * Path Parameter: `userId` (string, required) - The ID of the user to retrieve.
    * Response (Success 200): User object.
    * Response (Error 404): User with the given ID not found.
    * Example:
        ```bash
        curl http://localhost:8080/api/v1/users/charlie789
        ```

* **GET /users/{userId}/balance**
    * Description: Retrieves the current balance for a specific user.
    * Path Parameter: `userId` (string, required) - The ID of the user.
    * Response (Success 200):
        ```json
        {
            "user_id": "string",
            "balance": "float64"
        }
        ```
    * Response (Error 404): User with the given ID not found.
    * Example:
        ```bash
        curl http://localhost:8080/api/v1/users/charlie789/balance
        ```

* **PUT /users/{userId}**
    * Description: Updates details for a specific user. *(Note: The current implementation primarily updates the `updated_at` timestamp. Modify `UpdateUserRequest` and the service/repo layers to allow updating other fields like name if needed).*
    * Path Parameter: `userId` (string, required) - The ID of the user to update.
    * Request Body: `UpdateUserRequest` object (currently empty or for future fields).
        ```json
        {}
        ```
    * Response (Success 200): Updated user object.
    * Response (Error 400): Validation error on request body (if fields/validation added).
    * Response (Error 404): User with the given ID not found.
    * Example (assuming no updatable fields currently):
        ```bash
        curl -X PUT http://localhost:8080/api/v1/users/charlie789 \
        -H "Content-Type: application/json" \
        -d '{}'
        ```

* **DELETE /users/{userId}**
    * Description: Deletes a user by their ID. *(Note: Does not currently check for or handle associated bets).*
    * Path Parameter: `userId` (string, required) - The ID of the user to delete.
    * Response (Success 200): Confirmation message.
    * Response (Error 404): User with the given ID not found.
    * Example:
        ```bash
        curl -X DELETE http://localhost:8080/api/v1/users/charlie789
        ```

### Betting Operations

* **POST /bets**
    * Description: Places a new bet for a user on a specific event. Deducts the bet amount from the user's balance. Creates the user if they don't exist (with default balance before deduction). [cite: 2]
    * Request Body:
        ```json
        {
            "user_id": "string",  
            "event_id": "string", 
            "odds": "float64",      
            "amount": "float64"     
        }
        ```
    * Response (Success 201): The created bet object (including ID, status: PLACED, created_at).
    * Response (Error 400): Validation error (missing fields, invalid odds/amount, insufficient balance).
    * Response (Error 404): User creation failed (if applicable, should be rare with current logic).
    * Example:
        ```bash
        curl -X POST http://localhost:8080/api/v1/bets \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "alice123",
            "event_id": "match-xyz",
            "odds": 2.5,
            "amount": 100.0
        }'
        ```

* **POST /bets/settle/{eventId}**
    * Description: Settles all currently 'PLACED' bets associated with a specific event ID. Updates bet statuses to 'WON' or 'LOST' and adjusts user balances accordingly for winning bets. [cite: 2]
    * Path Parameter: `eventId` (string, required) - The ID of the event to settle.
    * Request Body:
        ```json
        {
            "result": "string"
        }
        ```
    * Response (Success 200): Confirmation message.
    * Response (Error 400): Invalid `result` value or missing `eventId`.
    * Response (Error 404): No 'PLACED' bets found for the given `eventId`.
    * Response (Error 409): If conflicts occur during update (e.g., a bet was already settled).
    * Response (Error 500): If errors occur during batch updates (potential partial success).
    * Example (Win):
        ```bash
        curl -X POST http://localhost:8080/api/v1/bets/settle/match-xyz \
        -H "Content-Type: application/json" \
        -d '{
            "result": "win"
        }'
        ```
    * Example (Lose):
        ```bash
        curl -X POST http://localhost:8080/api/v1/bets/settle/match-xyz \
        -H "Content-Type: application/json" \
        -d '{
            "result": "lose"
        }'
        ```
