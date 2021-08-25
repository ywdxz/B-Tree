package bt

type BT interface {
	Set(key int, value interface{})
	Del(key int)
	Get(key int) (value interface{}, ok bool)
	Print() (keyList []int, valueList []interface{})
}

type node struct {
	n int //关键字个数

	leaf bool //叶子节点标识

	c     []*node // t <= len(c) <= 2t
	key   []int   // 关键字,t-1 <= len(key) <= 2t-1
	value []interface{}
}

type bTree struct {
	root *node

	t int //最小度数,t>= 2
	//a. 除根节点外至少有t-1个关键字,t个孩子
	//b. 至多有2t-1个关键字,2t个孩子
}

func (b *bTree) search(x *node, key int) (*node, int) {

	i := 0
	for i < x.n && key > x.key[i] {
		i++
	}

	if i < x.n && key == x.key[i] {
		return x, i
	} else if x.leaf {
		//找不到
		return nil, 0
	} else {
		//Disk-Read(x,c[i])
		k := x.c[i]
		return b.search(k, key)
	}
}

func (b *bTree) splitChild(x *node, i int) {

	y := x.c[i]   //要分裂的节点
	y.n = b.t - 1 // 2t-1 => t-1

	z := &node{
		n:     y.n,
		key:   make([]int, 2*b.t-1),
		value: make([]interface{}, 2*b.t-1),
		c:     make([]*node, 2*b.t),
		leaf:  y.leaf,
	} //分裂出来的节点

	//copy y -> z
	for j := 0; j < b.t-1; j++ {
		z.key[j] = y.key[j+b.t]
		z.value[j] = y.value[j+b.t]
	}

	if !y.leaf {
		//不是叶子节点
		for i := 0; i < b.t; i++ {
			z.c[i] = y.c[i+b.t]
		}
	}

	//x节点空一个位置供y节点上升
	for j := x.n; j > i; j-- {
		x.c[j+1] = x.c[j]
	}

	x.c[i+1] = z
	for j := x.n - 1; j >= i; j-- {
		x.key[j+1] = x.key[j]
		x.value[j+1] = x.value[j]
	}
	x.key[i] = y.key[b.t-1]
	x.value[i] = y.value[b.t-1]

	x.n++

	//Disk-Write(x)
	//Disk-Write(y)
	//Disk-Write(z)
}

func (b *bTree) insert(k int, v interface{}) {

	r := b.root
	if r.n == 2*b.t-1 {
		s := &node{
			leaf:  false,
			n:     0,
			key:   make([]int, 2*b.t-1),
			value: make([]interface{}, 2*b.t-1),
			c:     make([]*node, 2*b.t),
		}
		b.root = s
		s.c[0] = r

		b.splitChild(s, 0)
		//Disk-read(s)
		b.insertNonfull(s, k, v)
	} else {
		b.insertNonfull(r, k, v)
	}

}

func (b *bTree) insertNonfull(x *node, k int, v interface{}) {

	i := x.n - 1

	if x.leaf {
		//叶子节点
		for i >= 0 && k < x.key[i] {
			x.key[i+1] = x.key[i]
			i--
		}
		i++

		x.key[i] = k
		x.value[i] = v
		x.n++
	} else {
		for i >= 0 && k < x.key[i] {
			i--
		}
		i++

		//Disk-Read(x.c[i])

		if x.c[i].n == 2*b.t-1 {
			//满节点
			b.splitChild(x, i)
			//Disk-read(x)
			if k > x.key[i] {
				i++
			}
		}

		b.insertNonfull(x.c[i], k, v)
	}
}

// mergeChild, y.n == t-1 & z.n == t-1, y < z
func (b *bTree) mergeChild(x *node, i int, y *node, z *node) {

	//y节点
	for j := b.t; j < 2*b.t-1; j++ {
		y.key[j] = z.key[j-b.t]
		y.value[j] = z.value[j-b.t]
	}
	y.key[b.t-1] = x.key[i]
	y.value[b.t-1] = x.value[i]

	if !y.leaf {
		//内部节点
		for j := b.t; j < 2*b.t; j++ {
			y.c[j] = z.c[j-b.t]
		}
	}
	y.n = 2*b.t - 1

	//x节点
	for j := i; j+1 < x.n; j++ {
		x.key[j] = x.key[j+1]
		x.value[j] = x.value[j+1]
	}

	for j := i + 1; j+1 < x.n+1; j++ {
		x.c[j] = x.c[j+1]
	}
	x.c[x.n] = nil
	x.n--

	//Disk-Write(x)
	//Disk-Write(y)
	//Disk-Write(z)
}

// borrowKey x: 父节点, x.key[i]: 当前key, y: 前一个节点, k: 当前节点, z: 后一个节点, return 是否成功
// 节点 k 向前后兄弟节点借一个key
func (b *bTree) borrowKey(x *node, i int, y *node, k *node, z *node) bool {

	if y != nil && y.n >= b.t {

		//k 节点
		for j := k.n - 1; j >= 0; j-- {
			k.key[j+1] = k.key[j]
			k.value[j+1] = k.value[j]
		}
		k.key[0] = x.key[i-1]
		k.value[0] = x.value[i-1]

		for j := k.n; j >= 0; j-- {
			k.c[j+1] = k.c[j]
		}
		k.c[0] = y.c[y.n]
		k.n++

		//x 节点
		x.key[i-1] = y.key[y.n-1]
		x.value[i-1] = y.value[y.n-1]

		//y 节点
		y.n--

		//Disk-Write(x)
		//Disk-Write(k)
		//Disk-Write(y)

		return true
	}

	if z != nil && z.n >= b.t {

		//k 节点
		k.key[k.n] = x.key[i]
		k.value[k.n] = x.value[i]
		k.c[k.n+1] = z.c[0]
		k.n++

		//x 节点
		x.key[i] = z.key[0]
		x.value[i] = z.value[0]

		//z 节点
		for j := 1; j < z.n; j++ {
			z.key[j-1] = z.key[j]
			z.value[j-1] = z.value[j]
		}
		for j := 1; j < z.n+1; j++ {
			z.c[j-1] = z.c[j]
		}
		z.c[z.n] = nil
		z.n--

		//Disk-Write(x)
		//Disk-Write(k)
		//Disk-Write(z)

		return true
	}

	return false
}

func (b *bTree) getMin(x *node) (z *node, k int) {

	for !x.leaf {
		//Disk-read(x)
		x = x.c[0]
	}
	z, k = x, 0
	return
}

func (b *bTree) getMax(x *node) (z *node, k int) {

	for !x.leaf {
		//Disk-read(x)
		x = x.c[x.n]
	}
	z, k = x, x.n-1
	return
}

func (b *bTree) deleteNonOne(x *node, key int) {

	if x.leaf {
		//叶子节点
		i := 0
		for i < x.n && x.key[i] < key {
			i++
		}

		if i == x.n || x.key[i] != key {
			//没找到
			return
		}

		for j := i; j+1 < x.n; j++ {
			x.key[j] = x.key[j+1]
			x.value[j] = x.value[j+1]
		}

		x.n--
		//Disk-write(x)
	} else {
		//内部节点
		i := 0
		for i < x.n && x.key[i] < key {
			i++
		}

		if i < x.n && key == x.key[i] {
			//找到了

			y := x.c[i]
			//Disk-read(y)
			z := x.c[i+1]
			//Disk-read(z)

			switch {
			case y.n >= b.t:
				//替换
				tmpNode, tmpIndex := b.getMax(y)
				x.key[i] = tmpNode.key[tmpIndex]
				x.value[i] = tmpNode.value[tmpIndex]
				b.deleteNonOne(y, tmpNode.key[tmpIndex])
			case z.n >= b.t:
				//替换
				tmpNode, tmpIndex := b.getMin(z)
				x.key[i] = tmpNode.key[tmpIndex]
				x.value[i] = tmpNode.value[tmpIndex]
				b.deleteNonOne(z, tmpNode.key[tmpIndex])
			default:
				//合并
				b.mergeChild(x, i, y, z)
				//Disk-read(y)
				b.deleteNonOne(y, key)
			}
		} else {
			//没到了

			k := x.c[i]
			//Disk-read(k)

			var y, z *node
			if i != 0 {
				y = x.c[i-1]
				//Disk-read(y)
			}

			if i != x.n {
				z = x.c[i+1]
				//Disk-read(z)
			}

			switch {
			case k.n >= b.t:
				b.deleteNonOne(k, key)
			case b.borrowKey(x, i, y, k, z): //尝试借
				b.deleteNonOne(k, key)
			default:
				if z != nil {
					//向后合并
					b.mergeChild(x, i, k, z)
					//Disk-read(k)
					b.deleteNonOne(k, key)
				} else {
					//向前合并
					b.mergeChild(x, i-1, y, k)
					//Disk-read(y)
					b.deleteNonOne(y, key)
				}
			}
		}
	}
}

func (b *bTree) delete(k int) {

	if b.root.n == 1 {

		y := b.root.c[0]
		//Disk-Read(y)
		z := b.root.c[1]
		//Disk-Read(z)

		switch {
		case b.root.leaf:
			b.deleteNonOne(b.root, k)
		case y.n == b.t-1 && z.n == b.t-1:
			b.mergeChild(b.root, 0, y, z)
			//Disk-Read(y)
			b.root = y
			b.deleteNonOne(y, k)
		default:
			b.deleteNonOne(b.root, k)
		}

	} else {
		b.deleteNonOne(b.root, k)
	}
}

func (b *bTree) print(x *node) (keyList []int, valueList []interface{}) {

	if x.leaf {
		keyList = append(keyList, x.key[0:x.n]...)
		valueList = append(valueList, x.value[0:x.n]...)
	} else {

		for i := 0; i < x.n; i++ {
			key, value := b.print(x.c[i])
			keyList = append(keyList, key...)
			valueList = append(valueList, value...)

			keyList = append(keyList, x.key[i])
			valueList = append(valueList, x.value[i])
		}

		key, value := b.print(x.c[x.n])
		keyList = append(keyList, key...)
		valueList = append(valueList, value...)
	}

	return
}

func GenBT(t int) BT {
	return &bTree{
		t: t,
		root: &node{
			leaf:  true,
			c:     make([]*node, 2*t),
			key:   make([]int, 2*t-1),
			value: make([]interface{}, 2*t-1),
		},
	}
}

func (b *bTree) Set(key int, value interface{}) {
	b.insert(key, value)
}

func (b *bTree) Del(key int) {
	b.delete(key)
}

func (b *bTree) Get(key int) (value interface{}, ok bool) {

	if node, index := b.search(b.root, key); node != nil {
		value, ok = node.value[index], true
	}
	return
}

func (b *bTree) Print() (keyList []int, valueList []interface{}) {
	keyList, valueList = b.print(b.root)
	return
}
