import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, CheckCircle2, ChevronRight, FileText, Lightbulb, LoaderCircle, ShieldCheck, Sparkles, ThumbsUp, UploadCloud, X } from 'lucide-react';
import { filesApi, resumeAnalysisApi } from '../shared/api/api';
import type { ResumeAnalysis, ResumeAnalysisResult } from '../shared/api/types';
import { ApiClientError } from '../shared/api/client';
import { Button, EmptyState, IconTile, LoadingState, Notice, PageHeader, StatusPill } from '../shared/ui';
import { formatDateTime } from '../shared/lib/dates';

function parseResult(raw: ResumeAnalysis['result']): ResumeAnalysisResult | null {
  if (!raw) return null;
  if (typeof raw === 'string') {
    try { return JSON.parse(raw) as ResumeAnalysisResult; } catch { return { summary: raw }; }
  }
  return raw;
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiClientError) {
    if (error.code === 'ANALYSIS_PROVIDER_UNAVAILABLE') return 'ИИ-анализ пока не настроен на сервере. Загрузка резюме доступна, а запись на консультацию продолжает работать.';
    if (error.code === 'FILE_TOO_LARGE') return 'Файл превышает допустимый размер 10 МБ.';
    if (error.code === 'INVALID_FILE_TYPE') return 'Файл должен быть в формате PDF или DOCX.';
    return error.message;
  }
  return error instanceof Error ? error.message : 'Не удалось выполнить анализ.';
}

export function ResumeAnalysisPage() {
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [aiConsent, setAiConsent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);
  const previous = useQuery({ queryKey: ['resume-analyses'], queryFn: resumeAnalysisApi.list });
  const detail = useQuery({
    queryKey: ['resume-analysis', activeId],
    queryFn: () => resumeAnalysisApi.get(activeId!),
    enabled: Boolean(activeId),
    refetchInterval: (query) => ['queued', 'processing'].includes(query.state.data?.status ?? '') ? 2500 : false,
  });
  const create = useMutation({
    mutationFn: async () => {
      if (!file) throw new Error('Сначала выбери файл резюме.');
      const saved = await filesApi.uploadResume(file);
      return resumeAnalysisApi.create(saved.id, aiConsent);
    },
    onSuccess: async (analysis) => {
      setActiveId(analysis.id);
      setFile(null);
      setAiConsent(false);
      setError(null);
      await queryClient.invalidateQueries({ queryKey: ['resume-analyses'] });
    },
    onError: (cause: unknown) => setError(errorMessage(cause)),
  });

  useEffect(() => {
    if (detail.data?.status === 'completed' || detail.data?.status === 'failed') {
      void queryClient.invalidateQueries({ queryKey: ['resume-analyses'] });
    }
  }, [detail.data?.status, queryClient]);

  const activeResult = useMemo(() => parseResult(detail.data?.result), [detail.data?.result]);
  const selectFile = (nextFile: File | undefined) => {
    setFileError(null); setError(null);
    if (!nextFile) { setFile(null); return; }
    const extension = nextFile.name.toLocaleLowerCase('ru-RU').split('.').pop();
    if (!['pdf', 'docx'].includes(extension ?? '')) { setFileError('Поддерживаются только PDF и DOCX.'); setFile(null); return; }
    if (nextFile.size > 10 * 1024 * 1024) { setFileError('Максимальный размер файла — 10 МБ.'); setFile(null); return; }
    if (nextFile.size === 0) { setFileError('Файл пустой. Выбери другой файл.'); setFile(null); return; }
    setFile(nextFile);
  };
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);
    if (!aiConsent) { setError('Подтверди согласие на обработку текста резюме внешним ИИ-провайдером перед запуском анализа.'); return; }
    create.mutate();
  };

  return <div className="page-stack">
    <PageHeader eyebrow="ИНСТРУМЕНТЫ ДЛЯ КАРЬЕРЫ" title="Анализ резюме с ИИ" description="Посмотри на своё резюме глазами рекрутера и получи идеи, как сделать его сильнее." />
    <div className="resume-intro"><div className="resume-intro-icon"><Sparkles size={28} /></div><div><h2>Сильное резюме — это не просто список навыков</h2><p>Загрузи резюме, а сервис проверит структуру, формулировки и полноту информации. Ты получишь рекомендации, которые можно применить к следующей версии.</p><div className="trust-row"><span><ShieldCheck size={15} /> Приватная загрузка</span><span><FileText size={15} /> PDF и DOCX</span><span><Lightbulb size={15} /> Конкретные рекомендации</span></div></div></div>

    <div className="resume-grid">
      <section className="panel-card resume-upload-card"><div className="panel-heading"><div><span className="eyebrow">ШАГ 1</span><h2>Загрузи резюме</h2><p>Файл останется доступен только с соответствующими правами.</p></div><IconTile icon={UploadCloud} tone="violet" /></div>
        <form onSubmit={submit}>
          <label className={`resume-dropzone ${file ? 'selected' : ''}`} htmlFor="analysis-file" onDragOver={(event) => event.preventDefault()} onDrop={(event) => { event.preventDefault(); selectFile(event.dataTransfer.files?.[0]); }}><input id="analysis-file" type="file" accept=".pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document" onChange={(event) => selectFile(event.target.files?.[0])} /><span className="upload-circle"><UploadCloud size={26} /></span><strong>{file ? file.name : 'Перетащи файл сюда'}</strong><span>{file ? `${(file.size / 1024 / 1024).toFixed(2)} МБ · файл выбран` : 'или нажми, чтобы выбрать на устройстве'}</span><span className="upload-badges"><b>PDF</b><b>DOCX</b><b>до 10 МБ</b></span></label>
          {file && <button type="button" className="clear-file" onClick={() => { setFile(null); const input = document.getElementById('analysis-file') as HTMLInputElement | null; if (input) input.value = ''; }}><X size={14} /> Убрать файл</button>}
          {fileError && <div className="field-error">{fileError}</div>}
          <label className="ai-consent-control"><input type="checkbox" checked={aiConsent} onChange={(event) => setAiConsent(event.target.checked)} /><span><strong>Согласие на ИИ-анализ</strong><small>Я согласен(на), что текст моего резюме будет передан внешнему ИИ-провайдеру для подготовки рекомендаций. Согласие сохраняется вместе с заданием анализа.</small></span></label>
          {error && <Notice tone="danger">{error}</Notice>}
          <Button type="submit" size="lg" className="full-width" disabled={!file || Boolean(fileError) || create.isPending}>{create.isPending ? <><LoaderCircle className="spin" size={17} /> Загружаем и запускаем…</> : <>Проверить резюме <ArrowRight size={17} /></>}</Button>
          <p className="privacy-note"><ShieldCheck size={15} /> Резюме передаётся на сервер. Анализ запускается только после явного согласия и при настроенном ИИ-провайдере.</p>
        </form>
      </section>

      <section className="panel-card analysis-result-card"><div className="panel-heading"><div><span className="eyebrow">ШАГ 2</span><h2>Результат проверки</h2><p>Здесь появится персональный отчёт.</p></div><IconTile icon={Sparkles} tone="green" /></div>
        {!activeId && <div className="result-placeholder"><div className="report-illustration"><div className="report-sheet"><span /><span /><span /><i><CheckCircle2 size={24} /></i></div><span className="report-spark">✦</span></div><strong>Твой отчёт появится здесь</strong><p>После загрузки резюме сервис начнёт обработку. Обычно статус обновляется автоматически.</p><div className="result-checklist"><span><CheckCircle2 size={16} /> Сильные стороны</span><span><CheckCircle2 size={16} /> Что улучшить</span><span><CheckCircle2 size={16} /> Следующие шаги</span></div></div>}
        {activeId && detail.isLoading && <LoadingState label="Загружаем состояние анализа…" />}
        {activeId && detail.isError && <Notice tone="danger">Не удалось получить состояние анализа. Проверь соединение и выбери запись в истории, чтобы повторить запрос.</Notice>}
        {detail.data && ['queued', 'processing'].includes(detail.data.status) && <div className="processing-state"><span className="processing-spinner"><LoaderCircle className="spin" size={31} /></span><h3>{detail.data.status === 'queued' ? 'Резюме в очереди' : 'Анализируем резюме'}</h3><p>{detail.data.status === 'queued' ? 'Как только обработчик освободится, проверка начнётся автоматически.' : 'Проверяем структуру и формулировки. Страница обновит статус сама.'}</p><div className="processing-steps"><span className="completed"><CheckCircle2 size={15} /> Файл загружен</span><span className="in-progress"><LoaderCircle size={15} /> Обработка</span><span><CheckCircle2 size={15} /> Отчёт</span></div></div>}
        {detail.data?.status === 'failed' && <div className="failed-state"><Notice tone="danger">Анализ не завершился. {detail.data.error_code ? `Код: ${detail.data.error_code}.` : 'Попробуй повторить позже.'}</Notice><Button variant="secondary" onClick={() => { setActiveId(null); setError('Выбери файл и запусти новый анализ.'); }}>Попробовать снова</Button></div>}
        {detail.data?.status === 'completed' && activeResult && <AnalysisReport result={activeResult} />}
        {detail.data?.status === 'completed' && !activeResult && <Notice tone="warning">Анализ завершился, но сервер не вернул содержимое отчёта.</Notice>}
      </section>
    </div>

    <section className="panel-card"><div className="panel-heading"><div><h2>История анализов</h2><p>Готовые отчёты можно открыть повторно</p></div><span className="count-label">{previous.data?.length ?? 0}</span></div>{previous.isLoading ? <LoadingState /> : previous.isError ? <Notice tone="danger">Не удалось загрузить историю анализов.</Notice> : (previous.data ?? []).length === 0 ? <EmptyState icon={FileText} title="Здесь пока пусто" description="После первой проверки история появится в этом разделе." /> : <div className="analysis-history">{previous.data!.map((item) => <button className={`analysis-history-row ${activeId === item.id ? 'active' : ''}`} key={item.id} onClick={() => { setActiveId(item.id); setError(null); }}><span className="history-file-icon"><FileText size={20} /></span><span className="analysis-history-main"><strong>Анализ резюме</strong><small>{formatDateTime(item.created_at)}</small></span><StatusPill status={item.status} /><ChevronRight size={18} className="analysis-chevron" /></button>)}</div>}</section>
  </div>;
}

function AnalysisReport({ result }: { result: ResumeAnalysisResult }) {
  const score = typeof result.score === 'number' ? result.score : null;
  const strengths = Array.isArray(result.strengths) ? result.strengths.filter((value): value is string => typeof value === 'string') : [];
  const weaknesses = Array.isArray(result.weaknesses) ? result.weaknesses.filter((value): value is string => typeof value === 'string') : [];
  const recommendations = Array.isArray(result.recommendations) ? result.recommendations.filter((value): value is string => typeof value === 'string') : [];
  const structureAdvice = Array.isArray(result.structure_advice) ? result.structure_advice.filter((value): value is string => typeof value === 'string') : [];
  const projectAdvice = Array.isArray(result.project_advice) ? result.project_advice.filter((value): value is string => typeof value === 'string') : [];
  const missingInformation = Array.isArray(result.missing_information) ? result.missing_information.filter((value): value is string => typeof value === 'string') : [];
  return <div className="report-content"><div className="report-summary"><div><span className="report-summary-label">ОБЩИЙ ВЫВОД</span><p>{result.summary || 'Отчёт готов. Изучи сильные стороны и рекомендации ниже.'}</p></div>{score !== null && <div className="score-ring" style={{ background: `conic-gradient(#697fe7 ${Math.max(0, Math.min(100, score)) * 3.6}deg, #e5e8fa ${Math.max(0, Math.min(100, score)) * 3.6}deg)` }}><div><strong>{score}</strong><span>из 100</span></div></div>}</div>
    {strengths.length > 0 && <div className="report-list-section"><h3><span className="report-section-icon good"><ThumbsUp size={16} /></span>Сильные стороны</h3><ul className="report-list good-list">{strengths.map((item, index) => <li key={`${index}-${item}`}><CheckCircle2 size={16} /><span>{item}</span></li>)}</ul></div>}
    {weaknesses.length > 0 && <div className="report-list-section"><h3><span className="report-section-icon warn"><Lightbulb size={16} /></span>Что стоит улучшить</h3><ul className="report-list warn-list">{weaknesses.map((item, index) => <li key={`${index}-${item}`}><span className="report-bullet" /><span>{item}</span></li>)}</ul></div>}
    {recommendations.length > 0 && <div className="report-list-section"><h3><span className="report-section-icon idea"><Sparkles size={16} /></span>Рекомендации</h3><ol className="report-recommendations">{recommendations.map((item, index) => <li key={`${index}-${item}`}><span>{String(index + 1).padStart(2, '0')}</span><p>{item}</p></li>)}</ol></div>}
    {structureAdvice.length > 0 && <div className="report-list-section"><h3><span className="report-section-icon idea"><FileText size={16} /></span>Структура резюме</h3><ul className="report-list">{structureAdvice.map((item, index) => <li key={`${index}-${item}`}><span className="report-bullet" /><span>{item}</span></li>)}</ul></div>}
    {projectAdvice.length > 0 && <div className="report-list-section"><h3><span className="report-section-icon good"><CheckCircle2 size={16} /></span>Проекты и достижения</h3><ul className="report-list good-list">{projectAdvice.map((item, index) => <li key={`${index}-${item}`}><CheckCircle2 size={16} /><span>{item}</span></li>)}</ul></div>}
    {missingInformation.length > 0 && <div className="report-list-section"><h3><span className="report-section-icon warn"><Lightbulb size={16} /></span>Что ещё добавить</h3><ul className="report-list warn-list">{missingInformation.map((item, index) => <li key={`${index}-${item}`}><span className="report-bullet" /><span>{item}</span></li>)}</ul></div>}
    {!strengths.length && !weaknesses.length && !recommendations.length && !structureAdvice.length && !projectAdvice.length && !missingInformation.length && <Notice tone="info">Отчёт содержит краткий вывод, но не все ожидаемые разделы. Посмотри итоговую рекомендацию выше.</Notice>}
  </div>;
}
