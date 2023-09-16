package game

type ItemDefinition struct {
    UniqueName  string
    Model       string
    ItemType    ItemType
    Radius      float64
    TurnsToLive int
}

type ItemType string

const (
    ItemTypeGrenade ItemType = "grenade" // direct reference for the gui icons asset names (TextureIndex: a.guiIcons[string(item.Definition.ItemType)])
)

type Item struct {
    Definition *ItemDefinition
}

func NewItem(definition *ItemDefinition) *Item {
    return &Item{
        Definition: definition,
    }
}
