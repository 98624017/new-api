/*
Copyright (C) 2025 QuantumNous

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

import React, { useEffect, useMemo, useState } from 'react';
import { Button, Empty, Input, Timeline, Typography } from '@douyinfe/semi-ui';
import { LockKeyhole, Megaphone } from 'lucide-react';
import { marked } from 'marked';
import { API } from '../../helpers';
import {
  unlockFrontendSession,
  verifyFrontendLockPassword,
} from '../../helpers/frontendLock';

const { Text, Title } = Typography;

const FrontendLock = ({ onUnlock }) => {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [noticeHtml, setNoticeHtml] = useState('');
  const [announcements, setAnnouncements] = useState([]);
  const [loadingNotice, setLoadingNotice] = useState(true);

  useEffect(() => {
    let mounted = true;

    async function loadNotice() {
      setLoadingNotice(true);
      const [noticeResult, statusResult] = await Promise.allSettled([
        API.get('/api/notice', { disableDuplicate: true }),
        API.get('/api/status', { disableDuplicate: true }),
      ]);

      if (!mounted) {
        return;
      }

      if (
        noticeResult.status === 'fulfilled' &&
        noticeResult.value?.data?.success &&
        noticeResult.value.data.data
      ) {
        setNoticeHtml(marked.parse(noticeResult.value.data.data));
      }

      if (
        statusResult.status === 'fulfilled' &&
        statusResult.value?.data?.success &&
        Array.isArray(statusResult.value.data.data?.announcements)
      ) {
        setAnnouncements(
          statusResult.value.data.data.announcements.slice(0, 5),
        );
      }

      setLoadingNotice(false);
    }

    loadNotice().catch(() => {
      if (mounted) {
        setLoadingNotice(false);
      }
    });

    return () => {
      mounted = false;
    };
  }, []);

  const processedAnnouncements = useMemo(() => {
    return announcements.map((item, index) => ({
      key: `${item?.publishDate || index}-${index}`,
      type: item?.type || 'default',
      time: item?.publishDate || '',
      contentHtml: marked.parse(item?.content || ''),
      extraHtml: item?.extra ? marked.parse(item.extra) : '',
    }));
  }, [announcements]);

  const handleSubmit = (event) => {
    event.preventDefault();
    if (!verifyFrontendLockPassword(password)) {
      setError('密码不正确');
      return;
    }

    unlockFrontendSession();
    setError('');
    onUnlock();
  };

  return (
    <main className='min-h-screen bg-semi-color-bg-0 text-semi-color-text-0'>
      <div className='mx-auto flex min-h-screen w-full max-w-5xl flex-col px-5 py-8 md:px-8'>
        <section className='grid flex-1 items-center gap-8 md:grid-cols-[0.9fr_1.1fr]'>
          <div className='space-y-6'>
            <div className='inline-flex h-12 w-12 items-center justify-center rounded-lg border border-semi-color-border bg-semi-color-fill-0'>
              <LockKeyhole size={24} />
            </div>
            <div className='space-y-3'>
              <Title heading={2} className='!m-0'>
                访问已锁定
              </Title>
              <Text type='tertiary'>
                请输入访问密码继续访问内部项目服务。本服务不对外提供访问。
              </Text>
            </div>

            <form className='max-w-sm space-y-3' onSubmit={handleSubmit}>
              <Input
                type='password'
                value={password}
                onChange={setPassword}
                placeholder='输入访问密码'
                size='large'
                autoFocus
              />
              {error && <Text type='danger'>{error}</Text>}
              <Button
                htmlType='submit'
                type='primary'
                theme='solid'
                size='large'
                block
              >
                解锁访问
              </Button>
            </form>
          </div>

          <div className='space-y-5'>
            <div className='border-b border-semi-color-border pb-3'>
              <div className='flex items-center gap-2'>
                <Megaphone size={18} />
                <Title heading={4} className='!m-0'>
                  站点公告
                </Title>
              </div>
            </div>

            {loadingNotice ? (
              <Empty description='公告加载中...' />
            ) : noticeHtml ? (
              <div
                className='notice-content-scroll max-h-72 overflow-y-auto pr-2 leading-7'
                dangerouslySetInnerHTML={{ __html: noticeHtml }}
              />
            ) : (
              <Empty description='暂无公告' />
            )}

            {processedAnnouncements.length > 0 && (
              <Timeline mode='left'>
                {processedAnnouncements.map((item) => (
                  <Timeline.Item
                    key={item.key}
                    type={item.type}
                    time={item.time}
                    extra={
                      item.extraHtml ? (
                        <div
                          className='text-xs text-semi-color-text-2'
                          dangerouslySetInnerHTML={{ __html: item.extraHtml }}
                        />
                      ) : null
                    }
                  >
                    <div
                      dangerouslySetInnerHTML={{ __html: item.contentHtml }}
                    />
                  </Timeline.Item>
                ))}
              </Timeline>
            )}
          </div>
        </section>
      </div>
    </main>
  );
};

export default FrontendLock;
