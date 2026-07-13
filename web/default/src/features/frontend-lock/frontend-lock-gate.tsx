/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ArrowRight, LockKeyhole, Megaphone } from 'lucide-react'
import { type FormEvent, type ReactNode, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'
import { getAnnouncementColorClass } from '@/lib/colors'
import {
  isFrontendLockEnabled,
  isFrontendLockUnlocked,
  unlockFrontendSession,
  verifyFrontendLockPassword,
} from '@/lib/frontend-lock'
import { cn } from '@/lib/utils'

interface FrontendLockGateProps {
  children: ReactNode
}

interface AnnouncementItem {
  id?: number | string
  type?: string
  content?: string
  extra?: string
  publishDate?: string
}

interface LockContent {
  notice: string
  announcements: AnnouncementItem[]
}

const emptyLockContent: LockContent = {
  notice: '',
  announcements: [],
}

function announcementKey(item: AnnouncementItem, index: number): string {
  if (item.id !== undefined && item.id !== null) return `id:${item.id}`
  return `${item.publishDate || 'announcement'}:${index}`
}

function LockAnnouncements(props: { content: LockContent; loading: boolean }) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <div className='space-y-5' aria-label={t('Loading...')}>
        <Skeleton className='h-5 w-2/3' />
        <Skeleton className='h-24 w-full' />
        <Skeleton className='h-16 w-full' />
      </div>
    )
  }

  if (!props.content.notice && props.content.announcements.length === 0) {
    return (
      <p className='text-muted-foreground border-border border-t py-6 text-sm'>
        {t('No announcements at this time')}
      </p>
    )
  }

  return (
    <ScrollArea className='max-h-[34rem] flex-1 pr-4 lg:max-h-[calc(100dvh-9rem)]'>
      {props.content.notice ? (
        <div className='pb-6 text-sm leading-7'>
          <RichContent breaks content={props.content.notice} />
        </div>
      ) : null}

      {props.content.notice && props.content.announcements.length > 0 ? (
        <Separator />
      ) : null}

      {props.content.announcements.length > 0 ? (
        <div className='divide-border divide-y'>
          {props.content.announcements.map((item, index) => (
            <article
              key={announcementKey(item, index)}
              className='flex gap-3 py-5'
            >
              <span
                aria-hidden='true'
                className={cn(
                  'mt-2 size-2 shrink-0 rounded-full',
                  getAnnouncementColorClass(item.type)
                )}
              />
              <div className='min-w-0 flex-1 space-y-2 text-sm'>
                <RichContent breaks content={item.content || ''} />
                {item.extra ? (
                  <RichContent
                    breaks
                    content={item.extra}
                    className='text-muted-foreground text-xs'
                  />
                ) : null}
                {item.publishDate ? (
                  <time className='text-muted-foreground block text-xs'>
                    {item.publishDate}
                  </time>
                ) : null}
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </ScrollArea>
  )
}

function FrontendLockScreen(props: { onUnlock: () => void }) {
  const { t } = useTranslation()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [content, setContent] = useState<LockContent>(emptyLockContent)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true

    void Promise.allSettled([
      api.get('/api/notice', {
        disableDuplicate: true,
        skipBusinessError: true,
        skipErrorHandler: true,
      }),
      api.get('/api/status', {
        disableDuplicate: true,
        skipBusinessError: true,
        skipErrorHandler: true,
      }),
    ])
      .then(([noticeResult, statusResult]) => {
        if (!active) return

        const notice =
          noticeResult.status === 'fulfilled' &&
          noticeResult.value.data?.success &&
          typeof noticeResult.value.data.data === 'string'
            ? noticeResult.value.data.data
            : ''
        const statusData =
          statusResult.status === 'fulfilled' &&
          statusResult.value.data?.success &&
          statusResult.value.data.data &&
          typeof statusResult.value.data.data === 'object'
            ? (statusResult.value.data.data as Record<string, unknown>)
            : null
        const announcements = Array.isArray(statusData?.announcements)
          ? (statusData.announcements as AnnouncementItem[]).slice(0, 5)
          : []

        setContent({ notice, announcements })
        setLoading(false)
      })
      .catch(() => {
        if (active) setLoading(false)
      })

    return () => {
      active = false
    }
  }, [])

  function handleSubmit(event: FormEvent<HTMLFormElement>): void {
    event.preventDefault()
    if (!verifyFrontendLockPassword(password)) {
      setError(t('Incorrect password'))
      return
    }

    unlockFrontendSession()
    setError('')
    props.onUnlock()
  }

  return (
    <main className='bg-background text-foreground min-h-dvh'>
      <div className='grid min-h-dvh lg:grid-cols-[minmax(24rem,0.82fr)_minmax(32rem,1.18fr)]'>
        <section className='border-border flex min-h-[60dvh] items-center border-b px-5 py-12 sm:px-10 lg:min-h-dvh lg:border-r lg:border-b-0 lg:px-14'>
          <div className='mx-auto w-full max-w-md lg:mx-0'>
            <div className='border-border bg-muted/40 mb-8 flex size-11 items-center justify-center rounded-lg border'>
              <LockKeyhole aria-hidden='true' className='size-5' />
            </div>

            <div className='mb-8 space-y-2'>
              <h1 className='text-2xl font-semibold sm:text-3xl'>
                {t('Access locked')}
              </h1>
              <p className='text-muted-foreground text-sm'>
                {t('Enter password')}
              </p>
            </div>

            <form className='space-y-4' onSubmit={handleSubmit}>
              <div className='space-y-2'>
                <Label htmlFor='frontend-lock-password'>{t('Password')}</Label>
                <Input
                  id='frontend-lock-password'
                  type='password'
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value)
                    if (error) setError('')
                  }}
                  autoComplete='current-password'
                  autoFocus
                  aria-invalid={Boolean(error)}
                  aria-describedby={error ? 'frontend-lock-error' : undefined}
                  className='h-10'
                />
                <p
                  id='frontend-lock-error'
                  aria-live='polite'
                  className='text-destructive min-h-5 text-sm'
                >
                  {error}
                </p>
              </div>

              <Button type='submit' size='lg' className='w-full'>
                {t('Continue')}
                <ArrowRight aria-hidden='true' data-icon='inline-end' />
              </Button>
            </form>
          </div>
        </section>

        <section className='bg-muted/15 flex min-h-[40dvh] flex-col px-5 py-10 sm:px-10 lg:min-h-dvh lg:px-14 lg:py-12'>
          <div className='mx-auto flex w-full max-w-2xl flex-1 flex-col'>
            <header className='mb-6 flex items-center gap-3'>
              <Megaphone
                aria-hidden='true'
                className='text-muted-foreground size-5'
              />
              <div>
                <h2 className='text-base font-semibold'>
                  {t('System Announcements')}
                </h2>
                <p className='text-muted-foreground text-xs'>
                  {t('Latest platform updates and notices')}
                </p>
              </div>
            </header>

            <LockAnnouncements content={content} loading={loading} />
          </div>
        </section>
      </div>
    </main>
  )
}

export function FrontendLockGate(props: FrontendLockGateProps) {
  const [locked, setLocked] = useState(
    () => isFrontendLockEnabled() && !isFrontendLockUnlocked()
  )

  if (locked) {
    return <FrontendLockScreen onUnlock={() => setLocked(false)} />
  }

  return props.children
}
