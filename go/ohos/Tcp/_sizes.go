package main
import ("fmt"; "unsafe"; . "gotest/ohos/Tcp")
func main() {
  var g StGlobal
  fmt.Println("StGlobal", unsafe.Sizeof(g))
  fmt.Println("  Sys", unsafe.Sizeof(g.Sys))
  fmt.Println("  grade", unsafe.Sizeof(g.grade))
  fmt.Println("  gexit", unsafe.Sizeof(g.gexit))
  fmt.Println("  gweight", unsafe.Sizeof(g.gweight))
  fmt.Println("  analogdensity", unsafe.Sizeof(g.analogdensity))
  fmt.Println("  exit[0]", unsafe.Sizeof(g.exit[0]))
  fmt.Println("  paras[0]", unsafe.Sizeof(g.paras[0]))
  fmt.Println("  weights[0]", unsafe.Sizeof(g.weights[0]))
  fmt.Println("  motor[0]", unsafe.Sizeof(g.motor[0]))
}
