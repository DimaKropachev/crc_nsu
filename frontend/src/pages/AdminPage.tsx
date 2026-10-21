import { useMemo, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CalendarDays, Check, CheckCircle2, Clock3, Download, Edit3, Plus, Save, ShieldCheck, SlidersHorizontal, UserRound, XCircle } from 'lucide-react';
import { adminApi, filesApi } from '../shared/api/api';
import type { Consultation, ConsultationSlot } from '../shared/api/types';
import { ApiClientError } from '../shared/api/client';
import { Avatar, Button, EmptyState, LoadingState, Notice, PageHeader, StatusPill } from '../shared/ui';
import { formatDateTime, formatTime, toDateTimeLocal } from '../shared/lib/dates';

type Tab = 'consultations' | 'schedule' | 'profile';

function errorText(error: unknown): string {
  if (error instanceof ApiClientError) return error.message;
  return error instanceof Error ? error.message : 'Произошла непредвиденная ошибка.';
}

export function AdminPage() {
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>('consultations');
  const [statusFilter, setStatusFilter] = useState('all');
  const from = useMemo(() => { const date = new Date(); date.setHours(0, 0, 0, 0); return date; }, []);
  const to = useMemo(() => { const date = new Date(from); date.setDate(date.getDate() + 179); return date; }, [from]);
  const consultations = useQuery({ queryKey: ['admin-consultations', statusFilter], queryFn: () => adminApi.consultations({ status: statusFilter === 'all' ? undefined : statusFilter, from, to }) });
  const slots = useQuery({ queryKey: ['admin-slots', from.toISOString(), to.toISOString()], queryFn: () => adminApi.slots(from, to) });
  const profile = useQuery({ queryKey: ['admin-profile'], queryFn: adminApi.profile });
  const records = consultations.data ?? [];
  const filteredSlots = (slots.data ?? []).sort((a, b) => a.starts_at.localeCompare(b.starts_at));
  const [generalError, setGeneralError] = useState<string | null>(null);
  const actions = useMutation({
    mutationFn: async ({ id, action }: { id: string; action: 'confirm' | 'complete' }) => action === 'confirm' ? adminApi.confirm(id) : adminApi.complete(id),
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['admin-consultations'] }); setGeneralError(null); },
    onError: (error: unknown) => setGeneralError(errorText(error)),
  });
  const blockSlot = useMutation({ mutationFn: adminApi.blockSlot, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['admin-slots'] }); }, onError: (error: unknown) => setGeneralError(errorText(error)) });
  const createSlot = useMutation({ mutationFn: adminApi.createSlot, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['admin-slots'] }); setGeneralError(null); }, onError: (error: unknown) => setGeneralError(errorText(error)) });
  const updateSlot = useMutation({ mutationFn: ({ id, starts_at, ends_at }: { id: string; starts_at: string; ends_at: string }) => adminApi.updateSlot(id, { starts_at, ends_at }), onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['admin-slots'] }); }, onError: (error: unknown) => setGeneralError(errorText(error)) });
  const updateProfile = useMutation({ mutationFn: adminApi.updateProfile, onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ['admin-profile'] }); setGeneralError(null); }, onError: (error: unknown) => setGeneralError(errorText(error)) });

  const saveSlot = async (input: { id?: string; starts_at: string; ends_at: string }): Promise<ConsultationSlot> => {
    const start = new Date(input.starts_at);
    const end = new Date(input.ends_at);
    if (!Number.isFinite(start.getTime()) || !Number.isFinite(end.getTime()) || end <= start) {
      const validationError = new Error('Проверь дату и время: окончание должно быть позже начала.');
      setGeneralError(validationError.message);
      throw validationError;
    }
    const body = { starts_at: start.toISOString(), ends_at: end.toISOString() };
    try {
      const item = input.id
        ? await updateSlot.mutateAsync({ id: input.id, ...body })
        : await createSlot.mutateAsync(body);
      setGeneralError(null);
      return item;
    } catch (error) {
      setGeneralError(errorText(error));
      throw error;
    }
  };

  const downloadResume = async (fileId: string) => {
    try {
      const blob = await filesApi.download(fileId);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a'); anchor.href = url; const extension = blob.type === 'application/pdf' ? '.pdf' : blob.type.includes('wordprocessingml') ? '.docx' : ''; anchor.download = `resume-${fileId}${extension}`; document.body.appendChild(anchor); anchor.click(); anchor.remove(); URL.revokeObjectURL(url);
    } catch (error) { setGeneralError(errorText(error)); }
  };

  return <div className="page-stack">
    <PageHeader eyebrow="КАБИНЕТ КОНСУЛЬТАНТА" title="Консультации и расписание" description="Обрабатывай заявки студентов, открывай доступное время и обновляй профиль." />
    <div className="admin-tabs" role="tablist" aria-label="Разделы кабинета"><button role="tab" aria-selected={tab === 'consultations'} className={tab === 'consultations' ? 'active' : ''} onClick={() => setTab('consultations')}><CalendarDays size={17} /> Консультации <span>{records.filter((item) => item.status === 'pending').length}</span></button><button role="tab" aria-selected={tab === 'schedule'} className={tab === 'schedule' ? 'active' : ''} onClick={() => setTab('schedule')}><Clock3 size={17} /> Расписание</button><button role="tab" aria-selected={tab === 'profile'} className={tab === 'profile' ? 'active' : ''} onClick={() => setTab('profile')}><UserRound size={17} /> Мой профиль</button></div>
    {generalError && <Notice tone="danger">{generalError}</Notice>}
    {tab === 'consultations' && <section className="panel-card"><div className="panel-heading"><div><h2>Заявки на консультацию</h2><p>Подтверждай новые записи и отмечай завершённые встречи.</p></div><label className="filter-select"><SlidersHorizontal size={15} /><select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)} aria-label="Фильтр по статусу"><option value="all">Все статусы</option><option value="pending">Ожидают подтверждения</option><option value="confirmed">Подтверждены</option><option value="completed">Завершены</option><option value="cancelled">Отменены</option></select></label></div>
      {consultations.isLoading ? <LoadingState /> : consultations.isError ? <Notice tone="danger">Не удалось загрузить консультации. {errorText(consultations.error)}</Notice> : records.length === 0 ? <EmptyState icon={CalendarDays} title="Заявок пока нет" description="Новые записи студентов появятся здесь." /> : <div className="admin-consultation-list">{records.map((item) => <AdminConsultationRow key={item.id} item={item} onConfirm={() => actions.mutate({ id: item.id, action: 'confirm' })} onComplete={() => actions.mutate({ id: item.id, action: 'complete' })} onDownload={downloadResume} busy={actions.isPending} />)}</div>}
    </section>}
    {tab === 'schedule' && <ScheduleManager slots={filteredSlots} loading={slots.isLoading} error={slots.error ? errorText(slots.error) : null} onCreate={saveSlot} onBlock={(id) => blockSlot.mutate(id)} onUpdate={saveSlot} busy={createSlot.isPending || updateSlot.isPending || blockSlot.isPending} />}
    {tab === 'profile' && <ProfileManager profile={profile.data} loading={profile.isLoading} error={profile.error ? errorText(profile.error) : null} saving={updateProfile.isPending} onSave={(input) => updateProfile.mutateAsync(input)} />}
  </div>;
}

function AdminConsultationRow({ item, onConfirm, onComplete, onDownload, busy }: { item: Consultation; onConfirm: () => void; onComplete: () => void; onDownload: (fileId: string) => void; busy: boolean }) {
  return <article className="admin-consultation-row"><div className="admin-date-icon"><strong>{new Date(item.starts_at).getDate()}</strong><span>{new Intl.DateTimeFormat('ru-RU', { month: 'short' }).format(new Date(item.starts_at)).replace('.', '')}</span></div><div className="admin-consultation-content"><div className="admin-consultation-top"><div><h3>{item.student_name || 'Студент НГУ'}</h3><p>{formatDateTime(item.starts_at)} · {formatTime(item.starts_at)}–{formatTime(item.ends_at)}</p></div><StatusPill status={item.status} /></div><div className="student-query"><span>ЗАПРОС СТУДЕНТА</span><p>{item.problem_description || 'Студент не добавил описание вопроса.'}</p></div><div className="admin-row-actions">{item.resume_file_id && <Button variant="secondary" size="sm" onClick={() => onDownload(item.resume_file_id!)}><Download size={15} /> Скачать резюме</Button>}{item.status === 'pending' && <Button size="sm" disabled={busy} onClick={onConfirm}><Check size={15} /> Подтвердить</Button>}{item.status === 'confirmed' && <Button size="sm" disabled={busy} onClick={onComplete}><CheckCircle2 size={15} /> Завершить</Button>}</div></div></article>;
}

function ScheduleManager({ slots, loading, error, onCreate, onBlock, onUpdate, busy }: { slots: ConsultationSlot[]; loading: boolean; error: string | null; onCreate: (input: { starts_at: string; ends_at: string; id?: string }) => Promise<unknown>; onBlock: (id: string) => void; onUpdate: (input: { starts_at: string; ends_at: string; id?: string }) => Promise<unknown>; busy: boolean }) {
  const [start, setStart] = useState('');
  const [end, setEnd] = useState('');
  const [editing, setEditing] = useState<string | null>(null);
  const [editStart, setEditStart] = useState('');
  const [editEnd, setEditEnd] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const create = async (event: FormEvent) => { event.preventDefault(); setFormError(null); if (!start || !end) { setFormError('Укажи начало и окончание слота.'); return; } if (new Date(start) <= new Date()) { setFormError('Нельзя создать слот в прошлом.'); return; } try { await onCreate({ starts_at: start, ends_at: end }); setStart(''); setEnd(''); } catch { /* The page-level error notice explains why saving failed. */ } };
  const edit = (slot: ConsultationSlot) => { setEditing(slot.id); setEditStart(toDateTimeLocal(new Date(slot.starts_at))); setEditEnd(toDateTimeLocal(new Date(slot.ends_at))); };
  const saveEdit = async (event: FormEvent) => { event.preventDefault(); if (!editing) return; try { await onUpdate({ id: editing, starts_at: editStart, ends_at: editEnd }); setEditing(null); } catch { /* Keep the editor open so the user can correct the values. */ } };
  return <div className="schedule-layout"><section className="panel-card"><div className="panel-heading"><div><h2>Мои слоты</h2><p>Управляй доступностью на ближайшие 6 месяцев.</p></div><span className="count-label">{slots.length} слотов</span></div>{loading ? <LoadingState /> : error ? <Notice tone="danger">{error}</Notice> : slots.length === 0 ? <EmptyState icon={CalendarDays} title="Расписание свободно" description="Создай первый слот для записи студентов." /> : <div className="slot-management-list">{slots.map((slot) => <div className="slot-management-row" key={slot.id}>{editing === slot.id ? <form className="slot-edit-form" onSubmit={saveEdit}><label>Начало<input type="datetime-local" value={editStart} onChange={(event) => setEditStart(event.target.value)} required /></label><label>Окончание<input type="datetime-local" value={editEnd} onChange={(event) => setEditEnd(event.target.value)} required /></label><div><Button size="sm" type="submit" disabled={busy}><Save size={14} /> Сохранить</Button><Button size="sm" variant="ghost" type="button" onClick={() => setEditing(null)}>Отмена</Button></div></form> : <><div className="slot-time-info"><CalendarDays size={17} /><div><strong>{formatDateTime(slot.starts_at)}</strong><span>{formatTime(slot.starts_at)}–{formatTime(slot.ends_at)}</span></div></div><StatusPill status={slot.status} /><div className="slot-management-actions">{slot.status === 'available' && <><button className="icon-button" aria-label="Изменить слот" onClick={() => edit(slot)}><Edit3 size={16} /></button><button className="icon-button danger-icon" aria-label="Заблокировать слот" onClick={() => { if (window.confirm('Заблокировать этот слот?')) onBlock(slot.id); }}><XCircle size={16} /></button></>}</div></>}</div>)}</div>}</section>
    <aside className="panel-card create-slot-card"><div className="create-slot-icon"><Plus size={22} /></div><h2>Добавить свободное время</h2><p>Студенты увидят этот слот в календаре и смогут выбрать его для консультации.</p><form className="vertical-form" onSubmit={create}><label>Дата и время начала<input className="form-control" type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} required /></label><label>Дата и время окончания<input className="form-control" type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)} required /></label>{formError && <div className="field-error">{formError}</div>}<Button size="lg" type="submit" className="full-width" disabled={busy}><Plus size={17} /> Создать слот</Button></form><div className="slot-note"><ShieldCheck size={15} /> Сервер проверит пересечение с другими слотами и отклонит конфликт.</div></aside></div>;
}

function ProfileManager({ profile, loading, error, saving, onSave }: { profile?: { first_name: string; last_name: string; specialization: string; description: string; avatar_url: string | null }; loading: boolean; error: string | null; saving: boolean; onSave: (input: { specialization: string; description: string }) => Promise<unknown> }) {
  const [specialization, setSpecialization] = useState<string | null>(null);
  const [description, setDescription] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  if (loading) return <section className="panel-card"><LoadingState /></section>;
  if (error || !profile) return <Notice tone="danger">{error ?? 'Профиль консультанта не найден.'}</Notice>;
  const currentSpecialization = specialization ?? profile.specialization;
  const currentDescription = description ?? profile.description;
  const save = async (event: FormEvent) => { event.preventDefault(); try { await onSave({ specialization: currentSpecialization.trim(), description: currentDescription.trim() }); setSaved(true); } catch { setSaved(false); } };
  return <section className="panel-card profile-editor"><div className="profile-editor-head"><Avatar name={`${profile.first_name} ${profile.last_name}`} src={profile.avatar_url} size="lg" /><div><h2>{profile.first_name} {profile.last_name}</h2><p>Публичная информация для студентов</p><span className="verified-label"><ShieldCheck size={14} /> Профиль консультанта</span></div></div>{saved && <Notice tone="success">Профиль успешно обновлён.</Notice>}<form className="vertical-form" onSubmit={save}><label>Специализация<input className="form-control" maxLength={200} value={currentSpecialization} onChange={(event) => { setSpecialization(event.target.value); setSaved(false); }} required placeholder="Например: IT-карьера и развитие" /></label><label>Описание<textarea className="form-control textarea-control" rows={5} maxLength={3000} value={currentDescription} onChange={(event) => { setDescription(event.target.value); setSaved(false); }} placeholder="Расскажи, с какими вопросами можешь помочь студентам…" /></label><div className="field-footnote"><span>Эта информация поможет студентам выбрать подходящего консультанта.</span><span>{currentDescription.length}/3000</span></div><div className="form-actions"><Button type="submit" size="lg" disabled={saving}>{saving ? 'Сохраняем…' : <><Save size={17} /> Сохранить изменения</>}</Button></div></form></section>;
}
