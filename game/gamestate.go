package game

type GameState interface {
    OnMouseMoved(oldX float64, oldY float64, newX float64, newY float64)
    OnDirectionKeys(movementVector [2]int, elapsed float64)
    OnMouseClicked(x float64, y float64)
}
