import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, CalendarDays, CheckCircle2, Clock3, FileText, RefreshCw, XCircle } from 'lucide-react';
import { consultationsApi, consultantsApi } from '../shared/api/api';
import type { Consultation, ConsultationSlot } from '../shared/api/types';
import { ApiClientError } from '../shared/api/client';
import { Button, EmptyState, LoadingState, Notice, PageHeader, StatusPill } from '../shared/ui';
import { formatDate, formatDateTime, formatTime } from '../shared/lib/dates';

function errorText(error: unknown): string {
  if (error instanceof ApiClientError) {
    if (error.code === 'RESCHEDULE_LIMIT_REACHED') return 'Ты уже переносил эту консультацию. Для нового времени нужно оформить новую запись.';
    if (error.code === 'SLOT_UNAVAILABLE') return 'Этот слот только что заняли. Выбери другой.';
    return error.message;
  }
  return error instanceof Error ? error.message : 'Не удалось выполнить действие.';
}

export function ConsultationsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const [pageNotice, setPageNotice] = useState(location.state?.bookingSuccess ? 'Запись создана! Консультант увидит заявку и сможет её подтвердить.' : null);
  const consultations = useQuery({ queryKey: ['consultations', 'mine'], queryFn: consultationsApi.mine });
  const active = (consultations.data ?? []).filter((item) => ['pending', 'confirmed'].includes(item.status));
  const past = (consultations.data ?? []).filter((item) => ['completed', 'cancelled'].includes(item.status));
  return <div className="page-stack">
    <PageHeader eyebrow="ТВОИ ВСТРЕЧИ" title="Мои консультации" description="Следи за статусом заявок и управляй запланированными встречами." action={<Button onClick={() => navigate('/booking')}><CalendarDays size={17} /> Новая запись</Button>} />
    {pageNotice && <Notice tone="success">{pageNotice}</Notice>}
    {consultations.isError && <Notice tone="danger">Не удалось загрузить твои консультации. {errorText(consultations.error)}</Notice>}
    <section className="panel-card"><div className="panel-heading"><div><h2>Предстоящие встречи</h2><p>Заявки, которые ещё не завершены</p></div><span className="count-label">{active.length} записей</span></div>
      {consultations.isLoading ? <LoadingState /> : active.length === 0 ? <EmptyState icon={CalendarDays} title="Пока нет запланированных встреч" description="Выбери консультанта и подходящее время — первая встреча появится здесь." action={<Button onClick={() => navigate('/booking')}>Записаться на консультацию</Button>} /> : <div className="consultation-list">{active.sort((a, b) => a.starts_at.localeCompare(b.starts_at)).map((item) => <ConsultationCard key={item.id} item={item} onUpdated={async () => { await Promise.all([queryClient.invalidateQueries({ queryKey: ['consultations', 'mine'] }), queryClient.invalidateQueries({ queryKey: ['consultant-slots'] })]); setPageNotice(null); }} />)}</div>}
    </section>
    {past.length > 0 && <section className="panel-card"><div className="panel-heading"><div><h2>История</h2><p>Завершённые и отменённые консультации</p></div><span className="count-label">{past.length}</span></div><div className="history-list">{past.sort((a, b) => b.starts_at.localeCompare(a.starts_at)).map((item) => <div className="history-row" key={item.id}><span className="history-icon"><CheckCircle2 size={18} /></span><div className="history-main"><strong>{item.consultant_name || 'Карьерный консультант'}</strong><span>{formatDateTime(item.starts_at)}</span></div><StatusPill status={item.status} /></div>)}</div></section>}
  </div>;
}

function ConsultationCard({ item, onUpdated }: { item: Consultation; onUpdated: () => Promise<void> }) {
  const queryClient = useQueryClient();
  const [rescheduling, setRescheduling] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<ConsultationSlot | null>(null);
  const [error, setError] = useState<string | null>(null);
  const from = new Date();
  from.setHours(0, 0, 0, 0);
  const to = new Date(from); to.setDate(to.getDate() + 60);
  const slots = useQuery({ queryKey: ['consultant-slots', item.consultant_id, from.toISOString(), to.toISOString()], queryFn: () => consultantsApi.slots(item.consultant_id, from, to), enabled: rescheduling });
  const reschedule = useMutation({ mutationFn: () => {
    if (!selectedSlot) throw new Error('Выбери новое время.');
    return consultationsApi.reschedule(item.id, selectedSlot.id);
  }, onSuccess: async () => { setRescheduling(false); setSelectedSlot(null); setError(null); await onUpdated(); }, onError: (cause: unknown) => setError(errorText(cause)) });
  const cancel = useMutation({ mutationFn: () => consultationsApi.cancel(item.id), onSuccess: async () => { await onUpdated(); }, onError: (cause: unknown) => setError(errorText(cause)) });
  const canReschedule = item.reschedule_count < 1 && new Date(item.starts_at) > new Date() && item.status !== 'cancelled';
  const canCancel = new Date(item.starts_at) > new Date();

  return <article className="consultation-card"><div className="consultation-card-main"><div className="consultation-date-icon"><CalendarDays size={22} /><span>{new Date(item.starts_at).getDate()}</span><small>{new Intl.DateTimeFormat('ru-RU', { month: 'short' }).format(new Date(item.starts_at)).replace('.', '')}</small></div><div className="consultation-details"><div className="consultation-heading"><div><h3>{item.consultant_name || 'Карьерный консультант'}</h3><div className="consultation-datetime"><Clock3 size={15} /> {formatDateTime(item.starts_at)} · {formatTime(item.starts_at)}–{formatTime(item.ends_at)}</div></div><StatusPill status={item.status} /></div><p>{item.problem_description || 'Карьерная консультация'}</p>{item.resume_file_id && <span className="attached-chip"><FileText size={14} /> Резюме прикреплено</span>}
    <div className="consultation-actions">{canReschedule && <Button variant="secondary" size="sm" onClick={() => { setRescheduling((value) => !value); setError(null); }}><RefreshCw size={15} /> {item.reschedule_count === 0 ? 'Перенести' : 'Перенос недоступен'}</Button>}{canCancel && <Button variant="ghost" size="sm" onClick={() => { if (window.confirm('Отменить эту консультацию?')) cancel.mutate(); }} disabled={cancel.isPending}><XCircle size={15} /> Отменить</Button>}<span className="reschedule-hint">{item.reschedule_count > 0 ? 'Перенос использован' : 'Можно перенести один раз'}</span></div>
    {error && <div className="inline-error">{error}</div>}
  </div></div>
  {rescheduling && <div className="reschedule-panel"><div className="panel-heading"><div><h3>Выбери новое время</h3><p>Текущая запись изменится только после подтверждения сервером.</p></div><button className="icon-button" aria-label="Закрыть перенос" onClick={() => setRescheduling(false)}><XCircle size={17} /></button></div>{slots.isLoading ? <LoadingState label="Загружаем свободные слоты…" /> : slots.isError ? <Notice tone="danger">Не удалось загрузить свободное время.</Notice> : (slots.data ?? []).length === 0 ? <p className="small-muted">Свободных слотов у этого консультанта пока нет.</p> : <div className="reschedule-slots">{slots.data!.filter((slot) => slot.id !== item.slot_id).map((slot) => <button key={slot.id} className={`slot-button ${selectedSlot?.id === slot.id ? 'selected' : ''}`} onClick={() => setSelectedSlot(slot)}><span>{formatDate(slot.starts_at)}</span><small>{formatTime(slot.starts_at)}</small></button>)}</div>}<div className="reschedule-footer"><Button variant="ghost" onClick={() => setRescheduling(false)}>Не менять</Button><Button disabled={!selectedSlot || reschedule.isPending} onClick={() => reschedule.mutate()}>{reschedule.isPending ? 'Переносим…' : 'Подтвердить перенос'} <ArrowRight size={16} /></Button></div></div>}
  </article>;
}
