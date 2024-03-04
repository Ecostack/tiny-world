package sys

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mlange-42/arche/ecs"
	"github.com/mlange-42/arche/generic"
	"github.com/mlange-42/tiny-world/game/comp"
	"github.com/mlange-42/tiny-world/game/res"
	"github.com/mlange-42/tiny-world/game/terr"
	"github.com/mlange-42/tiny-world/game/util"
)

// Build system.
type Build struct {
	rules           generic.Resource[res.Rules]
	view            generic.Resource[res.View]
	terrain         generic.Resource[res.Terrain]
	terrainEntities generic.Resource[res.TerrainEntities]
	landUse         generic.Resource[res.LandUse]
	landUseEntities generic.Resource[res.LandUseEntities]
	stock           generic.Resource[res.Stock]
	selection       generic.Resource[res.Selection]
	update          generic.Resource[res.UpdateInterval]
	ui              generic.Resource[res.UI]
	factory         generic.Resource[res.EntityFactory]

	radiusFilter    generic.Filter2[comp.Tile, comp.BuildRadius]
	warehouseFilter generic.Filter1[comp.Warehouse]
}

// Initialize the system
func (s *Build) Initialize(world *ecs.World) {
	s.rules = generic.NewResource[res.Rules](world)
	s.view = generic.NewResource[res.View](world)
	s.terrain = generic.NewResource[res.Terrain](world)
	s.terrainEntities = generic.NewResource[res.TerrainEntities](world)
	s.landUse = generic.NewResource[res.LandUse](world)
	s.landUseEntities = generic.NewResource[res.LandUseEntities](world)
	s.stock = generic.NewResource[res.Stock](world)
	s.selection = generic.NewResource[res.Selection](world)
	s.update = generic.NewResource[res.UpdateInterval](world)
	s.ui = generic.NewResource[res.UI](world)
	s.factory = generic.NewResource[res.EntityFactory](world)

	s.radiusFilter = *generic.NewFilter2[comp.Tile, comp.BuildRadius]()
	s.warehouseFilter = *generic.NewFilter1[comp.Warehouse]()
}

// Update the system
func (s *Build) Update(world *ecs.World) {
	if s.checkAbort() {
		return
	}
	sel := s.selection.Get()
	ui := s.ui.Get()
	view := s.view.Get()
	x, y := ebiten.CursorPosition()
	mx, my := view.ScreenToGlobal(x, y)
	cursor := view.GlobalToTile(mx, my)

	p := &terr.Properties[sel.BuildType]
	if p.TerrainBits.Contains(terr.CanBuy) && !util.IsBuildable(cursor.X, cursor.Y, s.radiusFilter.Query(world)) {
		return
	}

	fac := s.factory.Get()
	rules := s.rules.Get()
	stock := s.stock.Get()
	landUse := s.landUse.Get()
	landUseE := s.landUseEntities.Get()

	if sel.BuildType == terr.Bulldoze {
		luHere := landUse.Get(cursor.X, cursor.Y)
		luProps := &terr.Properties[luHere]

		if luProps.TerrainBits.Contains(terr.IsWarehouse) && s.isLastWarehouse(world) {
			return
		}

		if luProps.TerrainBits.Contains(terr.CanBuild) {
			world.RemoveEntity(landUseE.Get(cursor.X, cursor.Y))
			landUseE.Set(cursor.X, cursor.Y, ecs.Entity{})
			landUse.Set(cursor.X, cursor.Y, terr.Air)

			stock.Pay(p.BuildCost)
			ui.ReplaceButton(stock, rules)
		}
		return
	}

	if !stock.CanPay(p.BuildCost) {
		return
	}
	if p.Population > 0 && stock.Population+int(p.Population) > stock.MaxPopulation {
		return
	}

	terrain := s.terrain.Get()
	terrHere := terrain.Get(cursor.X, cursor.Y)
	if p.TerrainBits.Contains(terr.IsTerrain) {
		canBuild := p.BuildOn.Contains(terrHere) ||
			(sel.AllowRemove && terrHere != terr.Air && terrHere != sel.BuildType)
		if !canBuild {
			return
		}
		fac.Set(world, cursor.X, cursor.Y, sel.BuildType, sel.RandSprite)
	} else {
		if !p.BuildOn.Contains(terrHere) {
			return
		}

		luHere := landUse.Get(cursor.X, cursor.Y)
		luNatural := !terr.Properties[luHere].TerrainBits.Contains(terr.CanBuy)
		if luHere == terr.Air || (luNatural && p.TerrainBits.Contains(terr.CanBuy)) {
			if luHere != terr.Air {
				landUse.Set(cursor.X, cursor.Y, terr.Air)
				world.RemoveEntity(landUseE.Get(cursor.X, cursor.Y))
				landUseE.Set(cursor.X, cursor.Y, ecs.Entity{})
			}
			fac.Set(world, cursor.X, cursor.Y, sel.BuildType, sel.RandSprite)
		} else {
			return
		}
	}

	stock.Pay(p.BuildCost)
	ui.ReplaceButton(stock, rules)
}

// Finalize the system
func (s *Build) Finalize(world *ecs.World) {}

func (s *Build) checkAbort() bool {
	sel := s.selection.Get()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sel.Reset()
		return true
	}

	ui := s.ui.Get()
	x, y := ebiten.CursorPosition()
	if ui.MouseInside(x, y) {
		return true
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton2) {
		sel.Reset()
		return true
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		return true
	}

	p := &terr.Properties[sel.BuildType]
	if sel.BuildType != terr.Bulldoze && !p.TerrainBits.Contains(terr.CanBuild) {
		return true
	}
	return false
}

func (s *Build) isLastWarehouse(world *ecs.World) bool {
	query := s.warehouseFilter.Query(world)
	count := query.Count()
	query.Close()
	return count <= 1
}
