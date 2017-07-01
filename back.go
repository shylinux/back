package main // {{{
// }}}
import ( // {{{
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

// }}}
type Meta struct { // {{{
	flag rune
	name string
	time time.Time
	size int64
	hash string
}

// }}}
var ( // {{{
	ishelp  = flag.Bool("help", false, "usage: back [options] srcpath dstpath")
	istime  = flag.Bool("time", false, "compare files only by time stamp, don't use hash ")
	isforce = flag.Bool("force", false, "copy or move files without confirm ")

	issave   = flag.Bool("save", false, "copy new(+) and newer(>) files from srcpath to dstpath")
	isbackup = flag.Bool("backup", false, "copy all(+ > <) files from srcpath to dstpath")
	issame   = flag.Bool("same", false, "copy all(+ > <) files from srcpath to dstpath, move old(-) from dstpath to dstpath/trash")

	sumsize int64
	allsize int64
	src     string
	dst     string
)

// }}}

func sum(file string) string { // {{{
	if f, e := os.Open(file); e == nil {
		defer f.Close()

		h := md5.New()
		if _, e := io.Copy(h, f); e == nil {
			return hex.EncodeToString(h.Sum(nil))
		}
	}

	return ""
}

// }}}
func diff(srcmeta map[string]*Meta, dstmeta map[string]*Meta) { // {{{
	for k, v := range srcmeta {
		if vv, ok := dstmeta[k]; ok {
			if v.time.After(vv.time) {
				v.flag = '>'
				vv.flag = '>'
			} else if v.time.Before(vv.time) {
				v.flag = '<'
				vv.flag = '<'
			} else {
				v.flag = '='
				vv.flag = '='
			}

			if *istime {
				continue
			}

			if v.size == vv.size {
				v.hash = sum(path.Join(src, v.name))
				vv.hash = sum(path.Join(dst, vv.name))
				if v.hash == vv.hash {
					v.flag = '='
					vv.flag = '='
					continue
				}
			}

			if v.flag == '=' {
				v.flag = '>'
				vv.flag = '>'
			}
		} else {
			v.flag = '+'
		}
	}
}

// }}}
func scan(meta map[string]*Meta, file string, base string) (m map[string]*Meta, e error) { // {{{
	list, e := ioutil.ReadDir(file)
	if e != nil {
		return
	}

	for _, v := range list {
		if v.Name()[0] == '.' {
			continue
		}

		cwd := path.Join(file, v.Name())

		if v.IsDir() {
			if v.Name() != "." && v.Name() != ".." {
				if meta, e = scan(meta, cwd, base); e != nil {
					return nil, e
				}
			}
			continue
		}

		cwd = cwd[len(base):]

		m := new(Meta)
		m.flag = '-'

		m.name = fmt.Sprintf("%s", cwd)
		m.time = v.ModTime()

		m.size = v.Size()
		allsize += m.size

		meta[cwd] = m
		fmt.Print(".")
	}

	return meta, nil
}

// }}}

func sizes(s int64) string { // {{{
	if s > 10000000000 {
		return fmt.Sprintf("%4dG", s/1000000000)
	}
	if s > 10000000 {
		return fmt.Sprintf("%4dM", s/1000000)
	}
	if s > 10000 {
		return fmt.Sprintf("%4dK", s/1000)
	}
	return fmt.Sprintf("%4dB", s)
}

// }}}
func show(v *Meta) { // {{{
	fmt.Printf("%c %s %s %s\n", v.flag, v.time.Format("2006/01/02 15:04:05"), sizes(v.size), v.name)
}

// }}}
func save(size int64, src string, dst string) (e error) { // {{{

	dir := path.Dir(dst)
	if _, e = os.Stat(dir); e != nil {
		if e = os.MkdirAll(dir, os.ModePerm); e != nil {
			return e
		}
		fmt.Printf("%s create dst path %s\n", time.Now().Format("15:04:05"), dir)
	}

	for !*isforce {
		fmt.Printf("%s copy %s from %s to %s y(yes/no/quit/delete/compare):", time.Now().Format("15:04:05"), sizes(size), src, dst)

		var a string
		if fmt.Scanf("%s\n", &a); len(a) > 0 {
			switch a[0] {
			case 'n':
				return nil
			case 'q':
				os.Exit(0)
			case 'd':
				return os.Remove(src)
			case 'c':
				cmd := exec.Command("vim", "-d", src, dst)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				if cmd.Run() != nil {
					fmt.Println("can find editor vim")
				}
				continue
			}
		}
		break
	}

	var dstf, srcf *os.File
	if dstf, e = os.Create(dst); e == nil {
		defer dstf.Close()

		if srcf, e = os.Open(src); e == nil {
			defer srcf.Close()

			fmt.Printf("%s copy %s to %s ... ", time.Now().Format("15:04:05"), sizes(size), dst)

			size, e = io.Copy(dstf, srcf)
			sumsize += size

			fmt.Printf("done %%%d\n", sumsize*100/allsize)
		}
	}

	return e
}

// }}}
func confirm(format string, msg ...interface{}) bool { // {{{
	if *isforce {
		return true
	}

	fmt.Printf(format, msg...)

	var a string
	fmt.Scanf("%s\n", &a)
	if len(a) > 0 && a[0] == 'n' {
		return false
	}
	return true
}

// }}}
func action(done bool, srcmeta map[string]*Meta, dstmeta map[string]*Meta) (e error) { // {{{
	for _, v := range srcmeta {
		if v.flag == '+' {
			if *issame || *isbackup || *issave {
				if done {
					save(v.size, path.Join(src, v.name), path.Join(dst, v.name))
				} else {
					show(v)
					allsize += v.size
				}
			}
		}
	}

	for _, v := range srcmeta {
		if v.flag == '>' {
			if *issame || *isbackup || *issave {
				if done {
					save(v.size, path.Join(src, v.name), path.Join(dst, v.name))
				} else {
					show(v)
					allsize += v.size
				}
			}
		}
	}

	for _, v := range srcmeta {
		if v.flag == '<' {
			if *issame || *isbackup {
				if done {
					save(v.size, path.Join(src, v.name), path.Join(dst, v.name))
				} else {
					show(v)
					allsize += v.size
				}
			}
		}
	}

	for _, v := range dstmeta {
		if v.flag == '-' {
			if *issame {
				if done {
					trash := path.Join(dst, "trash")
					if _, e = os.Stat(trash); e != nil {
						os.MkdirAll(trash, 0776)
					}

					trash = path.Join(trash, fmt.Sprintf("%d-%s", time.Now().Unix(), path.Base(v.name)))

					if confirm("remove %s from %s y(yes/no):", v.name, dst) {
						if e = os.Rename(path.Join(dst, v.name), trash); e != nil {
							fmt.Printf("%s", e)
						}
					}
				} else {
					show(v)
				}
			}
		}
	}
	return
}

// }}}

func main() { // {{{
	var e error

	if flag.Parse(); flag.NArg() != 2 {
		fmt.Printf("usage %s [options] srcpath dstpath\n", os.Args[0])
		os.Exit(1)
	}

	old, _ := os.Getwd()
	if e = os.Chdir(flag.Arg(0)); e != nil {
		fmt.Printf("src %s : %s\n", flag.Arg(0), e)
		os.Exit(1)
	} else {
		src, _ = os.Getwd()
	}

	os.Chdir(old)
	dst = flag.Arg(1)
	if _, e = os.Stat(dst); e != nil {
		if e = os.MkdirAll(dst, os.ModePerm); e != nil {
			fmt.Printf("dst %s : %s\n", flag.Arg(0), e)
			os.Exit(1)
		} else {
			fmt.Printf("%s create dst path: %s\n", time.Now().Format("15:04:05"), dst)
		}
	}

	if e = os.Chdir(flag.Arg(1)); e != nil {
		fmt.Printf("dst %s : %s\n", flag.Arg(1), e)
		os.Exit(1)
	} else {
		dst, _ = os.Getwd()
	}

	allsize = 0
	fmt.Printf("%s sum src(%s): ", time.Now().Format("15:04:05"), src)
	srcmeta, e := scan(make(map[string]*Meta, 0), src, src)
	fmt.Printf(" %d files %s bytes\n", len(srcmeta), sizes(allsize))

	allsize = 0
	fmt.Printf("%s sum dst(%s): ", time.Now().Format("15:04:05"), dst)
	dstmeta, e := scan(make(map[string]*Meta, 0), dst, dst)
	fmt.Printf(" %d files %s bytes\n", len(dstmeta), sizes(allsize))

	diff(srcmeta, dstmeta)

	allsize = 0
	action(false, srcmeta, dstmeta)
	action(true, srcmeta, dstmeta)
}

// }}}
