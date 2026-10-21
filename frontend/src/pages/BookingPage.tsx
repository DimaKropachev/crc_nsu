import { useMemo, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, ArrowRight, CalendarDays, CheckCircle2, Clock3, FileText, Paperclip, UserRound, X } from 'lucide-react';
import { consultationsApi, consultantsApi, filesApi } from '../shared/api/api';
import type { ConsultationSlot, Consultant } from '../shared/api/types';
import { Avatar, Button, EmptyState, IconTile, LoadingState, Notice, PageHeader } from '../shared/ui';
import { formatDate, formatTime } from '../shared/lib/dates';
import { ApiClientError } from '../shared/api/client';

function messageForError(error: unknown): string {
  if (error instanceof ApiClientError) {
    if (error.code === 'SLOT_UNAVAILABLE') return 'Это время уже занял другой студент. Выбери другой слот.';
    if (error.code === 'FILE_TOO_LARGE') return 'Файл слишком большой. Максимальный размер — 10 МБ.';
    if (error.code === 'INVALID_FILE_TYPE') return 'Поддерживаются только PDF и DOCX.';
    return error.message;
  }
  return error instanceof Error ? error.message : 'Не удалось создать запись. Попробуй ещё раз.';
}

export function BookingPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [consultant, setConsultant] = useState<Consultant | null>(null);
  const [selectedSlot, setSelectedSlot] = useState<ConsultationSlot | null>(null);
  const [problem, setProblem] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const consultants = useQuery({ queryKey: ['consultants'], queryFn: consultantsApi.list });
  const from = useMemo(() => { const value = new Date(); value.setHours(0, 0, 0, 0); return value; }, []);
  const to = useMemo(() => { const value = new Date(from); value.setDate(value.getDate() + 60); return value; }, [from]);
  const slots = useQuery({ queryKey: ['consultant-slots', consultant?.id, from.toISOString(), to.toISOString()], queryFn: () => consultantsApi.slots(consultant!.id, from, to), enabled: Boolean(consultant) });

  const create = useMutation({
    mutationFn: async () => {
      if (!selectedSlot) throw new Error('Сначала выбери дату и время.');
      let resumeFileId: string | null = null;
      if (file) {
        const uploaded = await filesApi.uploadResume(file);
        resumeFileId = uploaded.id;
      }
      return consultationsApi.create({ slot_id: selectedSlot.id, problem_description: problem.trim(), resume_file_id: resumeFileId });
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['consultations', 'mine'] }),
        queryClient.invalidateQueries({ queryKey: ['consultant-slots'] }),
        queryClient.invalidateQueries({ queryKey: ['admin-consultations'] }),
      ]);
      navigate('/consultations', { state: { bookingSuccess: true } });
    },
    onError: (error: unknown) => setFormError(messageForError(error)),
  });

  const groupedSlots = (slots.data ?? []).filter((slot) => slot.status === 'available' && new Date(slot.starts_at) > new Date()).reduce<Record<string, ConsultationSlot[]>>((groups, slot) => {
    const key = new Date(slot.starts_at).toLocaleDateString('sv-SE');
    groups[key] = [...(groups[key] ?? []), slot];
    return groups;
  }, {});

  const selectFile = (nextFile: File | undefined) => {
    setFileError(null);
    if (!nextFile) { setFile(null); return; }
    const ext = nextFile.name.toLocaleLowerCase('ru-RU').split('.').pop();
    if (ext !== 'pdf' && ext !== 'docx') { setFileError('Выбери файл в формате PDF или DOCX.'); setFile(null); return; }
    if (nextFile.size > 10 * 1024 * 1024) { setFileError('Файл должен быть не больше 10 МБ.'); setFile(null); return; }
    setFile(nextFile);
  };

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    if (!selectedSlot) { setFormError('Выбери свободное время.'); return; }
    if (problem.trim().length < 5) { setFormError('Расскажи о своём вопросе чуть подробнее — минимум 5 символов.'); return; }
    create.mutate();
  };

  return <div className="page-stack">
    <PageHeader eyebrow="ЗАПИСЬ НА КОНСУЛЬТАЦИЮ" title="Найди свой следующий шаг" description="Выбери эксперта, удобное время и расскажи, с чем хочешь разобраться." action={<button className="back-link" onClick={() => navigate('/')}><ArrowLeft size={16} /> На главную</button>} />
    <div className="booking-steps"><div className={`booking-step ${!consultant ? 'current' : 'done'}`}><span>{consultant ? <CheckCircle2 size={18} /> : '01'}</span><div><strong>Консультант</strong><small>Выбери эксперта</small></div></div><div className="step-line" /><div className={`booking-step ${selectedSlot ? 'done' : consultant ? 'current' : ''}`}><span>{selectedSlot ? <CheckCircle2 size={18} /> : '02'}</span><div><strong>Время встречи</strong><small>Подбери слот</small></div></div><div className="step-line" /><div className={`booking-step ${selectedSlot ? 'current' : ''}`}><span>03</span><div><strong>Твой запрос</strong><small>Подтверди запись</small></div></div></div>

    <section className="booking-section"><div className="section-heading compact"><div><span className="eyebrow">ШАГ 1</span><h2>Выбери консультанта</h2></div><span className="count-label">{consultants.data?.length ?? 0} эксперта</span></div>
      {consultants.isLoading ? <LoadingState label="Загружаем консультантов…" /> : consultants.isError ? <Notice tone="danger">Не удалось получить список консультантов. Проверь доступность сервера и попробуй снова.</Notice> : !consultants.data?.length ? <EmptyState icon={UserRound} title="Пока нет активных консультантов" description="Попробуй вернуться позже." /> : <div className="consultant-select-grid">{consultants.data.map((item) => <button key={item.id} className={`consultant-select-card ${consultant?.id === item.id ? 'selected' : ''}`} onClick={() => { setConsultant(item); setSelectedSlot(null); }}><div className="consultant-card-heading"><Avatar name={`${item.first_name} ${item.last_name}`} src={item.avatar_url} size="lg" />{consultant?.id === item.id && <span className="selected-check"><CheckCircle2 size={18} /></span>}</div><strong>{item.first_name} {item.last_name}</strong><span className="consultant-specialty">{item.specialization}</span><p>{item.description || 'Поможет разобраться с карьерными вопросами и планом развития.'}</p><span className="select-expert">{consultant?.id === item.id ? 'Выбран' : 'Выбрать эксперта'} <ArrowRight size={15} /></span></button>)}</div>}
    </section>

    {consultant && <section className="booking-section"><div className="section-heading compact"><div><span className="eyebrow">ШАГ 2</span><h2>Подбери время</h2></div><span className="timezone-hint"><Clock3 size={14} /> Время указано для твоего устройства</span></div>
      {slots.isLoading ? <LoadingState label="Ищем свободное время…" /> : slots.isError ? <Notice tone="danger">Не удалось загрузить расписание. Обнови страницу или выбери другого консультанта.</Notice> : Object.keys(groupedSlots).length === 0 ? <EmptyState icon={CalendarDays} title="Свободных слотов пока нет" description="Попробуй выбрать другого консультанта или загляни позже." /> : <div className="slot-date-grid">{Object.entries(groupedSlots).sort(([a], [b]) => a.localeCompare(b)).map(([day, daySlots]) => <div className="slot-date-card" key={day}><div className="slot-date-label"><CalendarDays size={16} /><span>{formatDate(`${day}T00:00:00`)}</span><small>{daySlots.length} слота</small></div><div className="slot-buttons">{daySlots.sort((a, b) => a.starts_at.localeCompare(b.starts_at)).map((slot) => <button key={slot.id} className={`slot-button ${selectedSlot?.id === slot.id ? 'selected' : ''}`} onClick={() => setSelectedSlot(slot)}><span>{formatTime(slot.starts_at)}</span><small>{formatTime(slot.ends_at)}</small></button>)}</div></div>)}</div>}
    </section>}

    {selectedSlot && consultant && <section className="booking-section"><div className="section-heading compact"><div><span className="eyebrow">ШАГ 3</span><h2>Расскажи о своём запросе</h2></div></div><form className="booking-form panel-card" onSubmit={submit}>
      <div className="selected-summary"><div className="summary-people"><Avatar name={`${consultant.first_name} ${consultant.last_name}`} src={consultant.avatar_url} /><div><strong>{consultant.first_name} {consultant.last_name}</strong><span>{consultant.specialization}</span></div></div><div className="summary-time"><CalendarDays size={17} /><div><strong>{formatDate(selectedSlot.starts_at)}</strong><span>{formatTime(selectedSlot.starts_at)} — {formatTime(selectedSlot.ends_at)}</span></div><button type="button" className="icon-button" aria-label="Сменить время" onClick={() => { setSelectedSlot(null); document.querySelector('.booking-section:nth-of-type(2)')?.scrollIntoView({ behavior: 'smooth' }); }}><X size={16} /></button></div></div>
      <label className="form-label" htmlFor="problem">С чем хочешь разобраться? <span>*</span></label><textarea id="problem" className="form-control textarea-control" value={problem} onChange={(event) => setProblem(event.target.value)} minLength={5} maxLength={5000} required placeholder="Например: хочу подготовиться к собеседованию на стажировку Go-разработчика…" rows={4} /><div className="field-footnote"><span>Опиши цель или вопрос — это поможет консультанту подготовиться.</span><span>{problem.length}/5000</span></div>
      <div className="form-divider" /><div className="file-upload-heading"><div><label className="form-label" htmlFor="resume-file">Прикрепить резюме <span className="optional-label">необязательно</span></label><p>PDF или DOCX, до 10 МБ</p></div><IconTile icon={Paperclip} tone="violet" /></div>
      <label className={`file-picker ${file ? 'has-file' : ''}`} htmlFor="resume-file"><input id="resume-file" type="file" accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document" onChange={(event) => selectFile(event.target.files?.[0])} />{file ? <><span className="file-picker-icon"><FileText size={21} /></span><span className="file-picker-name"><strong>{file.name}</strong><small>{(file.size / (1024 * 1024)).toFixed(2)} МБ · Готов к загрузке</small></span><span className="file-remove" role="button" tabIndex={0} aria-label="Убрать файл" onClick={(event) => { event.preventDefault(); setFile(null); const element = document.getElementById('resume-file') as HTMLInputElement | null; if (element) element.value = ''; }}><X size={18} /></span></> : <><span className="file-picker-icon"><Paperclip size={21} /></span><span className="file-picker-name"><strong>Добавить файл резюме</strong><small>Нажми, чтобы выбрать файл на устройстве</small></span><span className="file-picker-add">+ Добавить</span></>}</label>
      {fileError && <div className="field-error">{fileError}</div>}{formError && <Notice tone="danger">{formError}</Notice>}
      <div className="booking-submit-row"><p>Нажимая кнопку, ты отправляешь заявку на карьерную консультацию.</p><Button size="lg" type="submit" disabled={create.isPending || Boolean(fileError)}>{create.isPending ? 'Отправляем…' : 'Подтвердить запись'} <ArrowRight size={17} /></Button></div>
    </form></section>}
  </div>;
}
