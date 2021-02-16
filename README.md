# Battleships

The poject is game, which is played by two players. Each player has playing field containing 10x10 fields. The rows are marked with letters A - J, the columns are marked with numbers 1 - 10. Each player has : 1 ship with length 5 fields, 2 ships with length 4 fields, 3 ships with length 3 fields and 4 ships with length 2 fields. In the begging, each player places his ships on his playing field. They can be placed horizontally or vertically in straigth line. Each player's goal is to hit all enemy ships. The players take turns and can shoot only once per turn. The player whose turn it is enters coordinates for the field that he wants to shoot at and receives indication abouth whether he has successfully hit an enemy ship and if yes whether he has sunk the ship. To sink ship the player has to hit all of his fields. The game ends when one player sinks all enemy ships.

## Game Server with the following functionality:

1. Creating room - create-room. Creates new room and returns the room ID.

2. List all active rooms - ls-rooms. Returns list of rooms containing tuples in the following format : roomID:playersCount. All possible values for playesrsCount are 1, 2. 1 - There is only one player in the room and the game hasn't started yet. 2 - All places in the room are taken and the game is in progress.

3. Join room by ID - join-room. Connects the player to the desired room. This will set him as Second to play and will notify the First player that he can make his turn. If the room doesn't exist or if it is already full the player will be notified with appropriate message.

4. Join random room - join-random. Searches for room with free place. If such room is found the player will join it. If there is no free room the player will receive appropriate message.

### During game
1. Ship placement - place. The player enters coordinates for the starting field of his ship x(A-J), y(0-9) and direction(up, down. left, right) in which the rest of the ship fields will be placed. The ship length is determined by the game.

2. Shooting at enemy field - shoot. The player enters coordinates x(A-J), y(0-9) of the field that he wants to attack. The player receives information whether he has hit the enemy ship and if yes whether he has sunk it. Ship is sunk if all his fields are destoyed.

3. Exit - exit. The player exits the room and his opponent wins the game. 

The server can run multiple games simultaneously.

## Client.

Simple console client which prompts the player to type in action or additional arguments for it.
