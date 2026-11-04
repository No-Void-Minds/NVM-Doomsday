package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/No-Void-Minds/nvm-doomsday/backend/internal/registration"
)

func registrationCmd(ctx context.Context, svc *registration.Service, args []string) error {
	switch args[0] {
	case "promote", "cancel":
		if len(args) != 2 {
			return errUsage
		}
		var ch registration.Change
		var err error
		if args[0] == "promote" {
			ch, err = svc.Promote(ctx, args[1])
		} else {
			ch, err = svc.Cancel(ctx, args[1])
		}
		if err != nil {
			return err
		}
		return printChange(ch)
	case "rename":
		return renameCmd(ctx, svc, args[1:])
	default:
		return errUsage
	}
}

// Kode dan posisi diambil dulu, baru flag dibaca, karena paket flag berhenti di argumen pertama yang bukan flag.
func renameCmd(ctx context.Context, svc *registration.Service, args []string) error {
	if len(args) < 3 {
		return errUsage
	}
	code := args[0]
	position, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("posisi harus angka: %w", err)
	}

	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	name := fs.String("name", "", "nama lengkap baru")
	email := fs.String("email", "", "email baru")
	account := fs.String("account", "", "nama akun baru")
	if err := fs.Parse(args[2:]); err != nil {
		return errUsage
	}

	// Hanya flag yang benar-benar ditulis yang dipakai, supaya field lain tidak ikut terkosongkan.
	var patch registration.AttendeePatch
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "name":
			patch.FullName = name
		case "email":
			patch.Email = email
		case "account":
			patch.AccountName = account
		}
	})
	if patch.FullName == nil && patch.Email == nil && patch.AccountName == nil {
		return errUsage
	}

	ch, err := svc.UpdateAttendee(ctx, code, position, patch)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s posisi %d\tsebelum\tsesudah\n", ch.Code, ch.Position)
	fmt.Fprintf(w, "nama lengkap\t%s\t%s\n", ch.Before.FullName, ch.After.FullName)
	fmt.Fprintf(w, "email\t%s\t%s\n", ch.Before.Email, ch.After.Email)
	fmt.Fprintf(w, "nama akun\t%s\t%s\n", ch.Before.AccountName, ch.After.AccountName)
	return w.Flush()
}

func printChange(ch registration.Change) error {
	b, a := ch.Before, ch.After
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "%s (%d orang)\tsebelum\tsesudah\n", ch.Code, b.Attendees)
	fmt.Fprintf(w, "status\t%s\t%s\n", b.Status, a.Status)
	fmt.Fprintf(w, "terkonfirmasi\t%d dari %d\t%d dari %d\n", b.Confirmed, b.Capacity, a.Confirmed, a.Capacity)
	fmt.Fprintf(w, "daftar tunggu\t%d\t%d\n", b.Waitlisted, a.Waitlisted)
	if err := w.Flush(); err != nil {
		return err
	}

	if ch.NextWaitlisted != nil {
		left := int64(a.Capacity) - a.Confirmed
		fmt.Printf("\nSisa %d kursi. Daftar tunggu teratas: %s (%d orang).\n",
			max(left, 0), ch.NextWaitlisted.Code, ch.NextWaitlisted.Attendees)
		fmt.Printf("Hubungi pendaftarnya, lalu jalankan: doomsday-admin promote %s\n", ch.NextWaitlisted.Code)
	}
	return nil
}
