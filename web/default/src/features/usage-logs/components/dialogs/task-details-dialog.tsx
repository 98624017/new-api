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
import {
  AlertTriangle,
  ChevronDown,
  ExternalLink,
  FileJson,
  Music,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { formatLogQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { TASK_ACTIONS, TASK_STATUS } from '../../constants'
import { formatDuration } from '../../lib/format'
import {
  taskActionMapper,
  taskPlatformMapper,
  taskStatusMapper,
} from '../../lib/mappers'
import {
  formatTaskPayload,
  getTaskAudioClips,
  getTaskFailureReason,
  parseTaskPayload,
} from '../../lib/task-details'
import type { TaskLog } from '../../types'
import { AudioPreviewDialog } from './audio-preview-dialog'

const VIDEO_ACTIONS = new Set<string>([
  TASK_ACTIONS.GENERATE,
  TASK_ACTIONS.TEXT_GENERATE,
  TASK_ACTIONS.FIRST_TAIL_GENERATE,
  TASK_ACTIONS.REFERENCE_GENERATE,
  TASK_ACTIONS.REMIX_GENERATE,
])
const OPENABLE_RESULT_URL_PATTERN = /^(https?:\/\/|\/v1\/videos\/)/i

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function DetailRow(props: {
  label: string
  value: React.ReactNode
  mono?: boolean
}) {
  return (
    <div className='grid min-w-0 gap-1 py-2.5 first:pt-0 last:pb-0 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-3'>
      <dt className='text-muted-foreground text-xs font-medium'>
        {props.label}
      </dt>
      <dd
        className={cn(
          'text-foreground min-w-0 text-sm break-words',
          props.mono && 'font-mono text-xs'
        )}
      >
        {props.value}
      </dd>
    </div>
  )
}

function DetailSection(props: {
  title: string
  icon?: React.ReactNode
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <section className='flex min-w-0 flex-col gap-2.5 border-t pt-4 first:border-t-0 first:pt-0'>
      <div className='flex min-w-0 items-center justify-between gap-2'>
        <h3 className='flex min-w-0 items-center gap-2 text-sm font-semibold'>
          {props.icon}
          <span className='truncate'>{props.title}</span>
        </h3>
        {props.action}
      </div>
      {props.children}
    </section>
  )
}

function PayloadBlock(props: { value: string; label: string }) {
  const { t } = useTranslation()

  return (
    <div className='relative min-w-0'>
      <CopyButton
        value={props.value}
        tooltip={t('Copy to clipboard')}
        className='bg-background/80 absolute top-2 right-2 z-10 size-7'
        iconClassName='size-3.5'
        aria-label={t('Copy {{label}}', { label: props.label })}
      />
      <pre className='bg-muted/40 max-h-80 min-w-0 overflow-auto rounded-md border p-3 pr-11 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap'>
        {props.value}
      </pre>
    </div>
  )
}

interface TaskDetailsDialogProps {
  log: TaskLog
  isAdmin: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function TaskDetailsDialog(props: TaskDetailsDialogProps) {
  const { t } = useTranslation()
  const [rawOpen, setRawOpen] = useState(false)
  const [audioOpen, setAudioOpen] = useState(false)

  const parsedProperties = useMemo(
    () => parseTaskPayload(props.log.properties),
    [props.log.properties]
  )
  const parsedData = useMemo(
    () => parseTaskPayload(props.log.data),
    [props.log.data]
  )
  const propertiesText = useMemo(
    () => formatTaskPayload(props.log.properties),
    [props.log.properties]
  )
  const upstreamResponse = useMemo(
    () => formatTaskPayload(props.log.data),
    [props.log.data]
  )
  const rawTaskData = useMemo(
    () => (rawOpen ? formatTaskPayload(props.log) : ''),
    [props.log, rawOpen]
  )
  const audioClips = useMemo(
    () => getTaskAudioClips(props.log.data),
    [props.log.data]
  )

  let model = ''
  if (isRecord(parsedProperties)) {
    const originModel = parsedProperties.origin_model_name
    const upstreamModel = parsedProperties.upstream_model_name
    if (typeof originModel === 'string') model = originModel
    else if (typeof upstreamModel === 'string') model = upstreamModel
  }
  if (!model && isRecord(parsedData) && typeof parsedData.model === 'string') {
    model = parsedData.model
  }

  const duration = formatDuration(
    props.log.submit_time,
    props.log.finish_time,
    'seconds'
  )
  const isVideoTask = VIDEO_ACTIONS.has(props.log.action)
  const videoUrl =
    isVideoTask && props.log.status === TASK_STATUS.SUCCESS && props.log.task_id
      ? `/v1/videos/${props.log.task_id}/content`
      : ''
  const resultUrl = props.log.result_url?.trim() ?? ''
  const failureReason = getTaskFailureReason(
    props.log.status,
    props.log.fail_reason,
    resultUrl
  )
  const openableResultUrl = OPENABLE_RESULT_URL_PATTERN.test(resultUrl)
    ? resultUrl
    : ''
  const resultLink = videoUrl || openableResultUrl
  const hasDistinctResultUrl = Boolean(
    resultUrl && resultUrl !== props.log.fail_reason
  )
  const hasResult = Boolean(
    hasDistinctResultUrl || resultLink || audioClips.length
  )

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={t('Task Details')}
        description={t('View complete task metadata and upstream response')}
        contentClassName='max-h-[calc(100dvh-1rem)] max-sm:w-[calc(100%-1rem)] max-sm:max-w-none sm:max-w-3xl'
        contentHeight='min(72vh, 760px)'
        bodyClassName='flex flex-col gap-4 py-1'
        footer={
          <Button
            variant='outline'
            onClick={() => props.onOpenChange(false)}
            className='w-full sm:w-auto'
          >
            {t('Close')}
          </Button>
        }
      >
        <DetailSection title={t('Task Information')}>
          <dl className='divide-border divide-y'>
            <DetailRow
              label={t('Task ID')}
              mono
              value={
                <span className='flex min-w-0 items-center gap-1.5'>
                  <span className='min-w-0 break-all'>{props.log.task_id}</span>
                  <CopyButton
                    value={props.log.task_id}
                    tooltip={t('Copy to clipboard')}
                    className='size-7'
                    iconClassName='size-3.5'
                  />
                </span>
              }
            />
            <DetailRow
              label={t('Platform')}
              value={t(
                taskPlatformMapper.getLabel(
                  props.log.platform,
                  props.log.platform || 'Unknown'
                )
              )}
            />
            <DetailRow
              label={t('Action')}
              value={t(
                taskActionMapper.getLabel(
                  props.log.action,
                  props.log.action || 'Unknown'
                )
              )}
            />
            <DetailRow
              label={t('Status')}
              value={
                <StatusBadge
                  label={t(
                    taskStatusMapper.getLabel(
                      props.log.status,
                      props.log.status || 'Unknown'
                    )
                  )}
                  variant={taskStatusMapper.getVariant(props.log.status)}
                  copyable={false}
                  size='sm'
                />
              }
            />
            {props.log.progress ? (
              <DetailRow label={t('Progress')} value={props.log.progress} />
            ) : null}
            {model ? <DetailRow label={t('Model')} value={model} mono /> : null}
          </dl>
        </DetailSection>

        <DetailSection title={t('Timing')}>
          <dl className='divide-border divide-y'>
            <DetailRow
              label={t('Created At')}
              value={formatTimestampToDate(props.log.created_at, 'seconds')}
              mono
            />
            <DetailRow
              label={t('Updated At')}
              value={formatTimestampToDate(props.log.updated_at, 'seconds')}
              mono
            />
            <DetailRow
              label={t('Submit Time')}
              value={formatTimestampToDate(props.log.submit_time, 'seconds')}
              mono
            />
            {props.log.start_time ? (
              <DetailRow
                label={t('Start Time')}
                value={formatTimestampToDate(props.log.start_time, 'seconds')}
                mono
              />
            ) : null}
            {props.log.finish_time ? (
              <DetailRow
                label={t('Finish Time')}
                value={formatTimestampToDate(props.log.finish_time, 'seconds')}
                mono
              />
            ) : null}
            {duration ? (
              <DetailRow
                label={t('Duration')}
                value={`${duration.durationSec.toFixed(1)}s`}
                mono
              />
            ) : null}
          </dl>
        </DetailSection>

        <DetailSection title={t('Billing')}>
          <dl className='divide-border divide-y'>
            <DetailRow
              label={t('Cost')}
              value={formatLogQuota(props.log.quota)}
              mono
            />
            {props.log.group ? (
              <DetailRow label={t('Group')} value={props.log.group} mono />
            ) : null}
            {props.isAdmin ? (
              <DetailRow
                label={t('Channel')}
                value={`#${props.log.channel_id}`}
                mono
              />
            ) : null}
            {props.isAdmin && props.log.username ? (
              <DetailRow label={t('User')} value={props.log.username} />
            ) : null}
            {props.isAdmin ? (
              <DetailRow
                label={t('User ID')}
                value={String(props.log.user_id)}
                mono
              />
            ) : null}
          </dl>
        </DetailSection>

        {propertiesText ? (
          <DetailSection title={t('Request Properties')}>
            <PayloadBlock
              value={propertiesText}
              label={t('Request Properties')}
            />
          </DetailSection>
        ) : null}

        {failureReason ? (
          <DetailSection
            title={t('Fail Reason')}
            icon={<AlertTriangle className='text-destructive size-4' />}
          >
            <PayloadBlock value={failureReason} label={t('Fail Reason')} />
          </DetailSection>
        ) : null}

        {hasResult ? (
          <DetailSection title={t('Result')}>
            <div className='flex flex-col gap-3'>
              {hasDistinctResultUrl ? (
                <PayloadBlock value={resultUrl} label={t('Result URL')} />
              ) : null}
              <div className='flex flex-wrap gap-2'>
                {resultLink ? (
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() =>
                      window.open(resultLink, '_blank', 'noopener,noreferrer')
                    }
                  >
                    <ExternalLink data-icon='inline-start' />
                    {t('Open in new tab')}
                  </Button>
                ) : null}
                {audioClips.length ? (
                  <Button
                    variant='outline'
                    size='sm'
                    onClick={() => setAudioOpen(true)}
                  >
                    <Music data-icon='inline-start' />
                    {t('Audio Preview')}
                  </Button>
                ) : null}
              </div>
            </div>
          </DetailSection>
        ) : null}

        {upstreamResponse ? (
          <DetailSection title={t('Upstream Response')}>
            <PayloadBlock
              value={upstreamResponse}
              label={t('Upstream Response')}
            />
          </DetailSection>
        ) : null}

        <Collapsible open={rawOpen} onOpenChange={setRawOpen}>
          <DetailSection
            title={t('Raw Task Data')}
            icon={<FileJson className='text-muted-foreground size-4' />}
            action={
              <CollapsibleTrigger
                render={
                  <Button
                    type='button'
                    variant='ghost'
                    size='sm'
                    className='h-7 gap-1 px-2 text-xs'
                  />
                }
              >
                {rawOpen ? t('Hide') : t('Show')}
                <ChevronDown
                  data-icon='inline-end'
                  className={cn(
                    'transition-transform',
                    rawOpen && 'rotate-180'
                  )}
                />
              </CollapsibleTrigger>
            }
          >
            <CollapsibleContent>
              <PayloadBlock value={rawTaskData} label={t('Raw Task Data')} />
            </CollapsibleContent>
          </DetailSection>
        </Collapsible>
      </Dialog>

      <AudioPreviewDialog
        open={audioOpen}
        onOpenChange={setAudioOpen}
        clips={audioClips}
      />
    </>
  )
}
