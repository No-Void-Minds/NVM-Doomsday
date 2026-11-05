import pg from 'pg'

import { testDatabaseUrls } from './env'

// Login sebagai doomsday_migrator, karena doomsday_app tidak punya izin DELETE atau TRUNCATE.
// testDatabaseUrls sudah menolak database yang tidak berakhiran _test.
export class TestDb {
  private readonly url = testDatabaseUrls().migrator

  async query<T extends pg.QueryResultRow = Record<string, unknown>>(
    sql: string,
    params: unknown[] = [],
  ): Promise<T[]> {
    const client = new pg.Client({ connectionString: this.url })
    await client.connect()
    try {
      return (await client.query<T>(sql, params)).rows
    } finally {
      await client.end()
    }
  }

  async count(sql: string, params: unknown[] = []): Promise<number> {
    const rows = await this.query<{ n: string }>(`SELECT (${sql}) AS n`, params)
    return Number(rows[0]?.n)
  }

  // Sama dengan testdb.Reset di backend: data peserta dan VIP kosong, event_config kembali ke seed 0003.
  async reset(): Promise<void> {
    await this.query('TRUNCATE attendees, registrations, vip_accounts')
    await this.query(`
      INSERT INTO event_config (id, title, film_title, capacity, max_guests_vip, is_open)
      VALUES (1, 'No Void Minds Doomsday', 'Avengers: Doomsday', 200, 3, true)
      ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title, film_title = EXCLUDED.film_title,
        cinema_name = NULL, city = NULL, studio = NULL, gather_at = NULL, show_at = NULL, map_url = NULL,
        capacity = EXCLUDED.capacity, max_guests_vip = EXCLUDED.max_guests_vip, is_open = EXCLUDED.is_open,
        updated_at = now()`)
  }

  async setEvent(fields: {
    capacity?: number
    max_guests_vip?: number
    is_open?: boolean
    show_at?: string | null
    gather_at?: string | null
  }): Promise<void> {
    const entries = Object.entries(fields)
    const sets = entries.map(([k], i) => `${k} = $${i + 1}`).join(', ')
    await this.query(
      `UPDATE event_config SET ${sets} WHERE id = 1`,
      entries.map(([, v]) => v),
    )
  }

  // Kunci dibentuk seperti NormalizeAccount untuk nama ASCII biasa yang dipakai di test.
  async addVip(...names: string[]): Promise<void> {
    for (const name of names) {
      await this.query(
        `INSERT INTO vip_accounts (account_name, account_key) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
        [name, name.trim().replace(/^@+/, '').trim().toLowerCase()],
      )
    }
  }

  async setVipActive(name: string, active: boolean): Promise<void> {
    await this.query('UPDATE vip_accounts SET is_active = $2 WHERE account_name = $1', [
      name,
      active,
    ])
  }
}
