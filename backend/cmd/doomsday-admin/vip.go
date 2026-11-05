package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/vip"
)

func vipCmd(ctx context.Context, svc *vip.Service, args []string) error {
	if len(args) == 0 {
		return errUsage
	}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return errUsage
		}
		return vipList(ctx, svc)
	case "add":
		if len(args) < 2 {
			return errUsage
		}
		return vipAdd(ctx, svc, args[1:])
	case "on", "off":
		if len(args) != 2 {
			return errUsage
		}
		acc, err := svc.SetActiveByName(ctx, args[1], args[0] == "on")
		if err != nil {
			return err
		}
		fmt.Printf("%s sekarang %s.\n", acc.Name, activeLabel(acc.Active))
		if !acc.Active {
			fmt.Println("Kalau nama ini sudah dipakai registrasi, registrasinya tetap VIP. Cek dengan: doomsday-admin vip list")
		}
		return nil
	default:
		return errUsage
	}
}

func vipList(ctx context.Context, svc *vip.Service) error {
	accounts, err := svc.List(ctx)
	if err != nil {
		return err
	}
	active := 0
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "nama akun\tstatus\tdipakai")
	for _, a := range accounts {
		if a.Active {
			active++
		}
		used := a.RegistrationCode
		if used == "" {
			used = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, activeLabel(a.Active), used)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Printf("\n%d nama, %d aktif.\n", len(accounts), active)
	return nil
}

func vipAdd(ctx context.Context, svc *vip.Service, names []string) error {
	rep, err := svc.Add(ctx, names)
	if err != nil {
		return err
	}
	printGroup("ditambah", rep.Added)
	printGroup("diaktifkan lagi", rep.Reactivated)
	printGroup("sudah ada", rep.Existing)
	for _, r := range rep.Rejected {
		fmt.Printf("ditolak: %q, %s\n", r.Name, r.Message)
	}
	return nil
}

func printGroup(label string, names []string) {
	if len(names) > 0 {
		fmt.Printf("%s: %s\n", label, strings.Join(names, ", "))
	}
}

func activeLabel(active bool) string {
	if active {
		return "aktif"
	}
	return "nonaktif"
}
