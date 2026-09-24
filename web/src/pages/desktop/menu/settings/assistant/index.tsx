import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { Button, Divider, Input, InputNumber, message, Select, Switch, Upload } from 'antd';
import { useSetAtom } from 'jotai';
import { Trash2Icon, UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/assistant.ts';
import type { AssistantConfig, AssistantConfigUpdate, AttachmentInfo } from '@/api/assistant.ts';
import { assistantConfigAtom } from '@/jotai/assistant.ts';

type SecretKey = 'geminiApiKey' | 'orApiKey' | 'proxyPass';
const emptySecrets: Record<SecretKey, string> = { geminiApiKey: '', orApiKey: '', proxyPass: '' };

const Row = ({ label, desc, children }: { label: string; desc?: string; children: ReactNode }) => (
  <div className="flex items-center justify-between gap-4 py-2">
    <div className="flex flex-col">
      <span>{label}</span>
      {desc && <span className="text-xs text-neutral-500">{desc}</span>}
    </div>
    <div className="flex shrink-0 items-center gap-2">{children}</div>
  </div>
);

export const AssistantSettings = () => {
  const { t } = useTranslation();
  const setRuntimeConfig = useSetAtom(assistantConfigAtom);
  const [config, setConfig] = useState<AssistantConfig | null>(null);
  const [secrets, setSecrets] = useState(emptySecrets);
  const [attachments, setAttachments] = useState<AttachmentInfo[]>([]);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    api.getAssistantConfig().then((rsp) => rsp.code === 0 && setConfig(rsp.data));
    loadAttachments();
  }, []);

  function loadAttachments() {
    api.listAttachments().then((rsp) => rsp.code === 0 && setAttachments(rsp.data));
  }

  function update<K extends keyof AssistantConfig>(key: K, value: AssistantConfig[K]) {
    setConfig((c) => (c ? { ...c, [key]: value } : c));
  }

  function applyResponse(rsp: { code: number; msg: string; data: AssistantConfig }) {
    if (rsp.code !== 0) {
      message.error(rsp.msg);
      return false;
    }
    setConfig(rsp.data);
    setRuntimeConfig(rsp.data);
    return true;
  }

  async function save() {
    if (!config) return;
    setIsSaving(true);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { hasGeminiApiKey, hasOrApiKey, hasProxyPass, ...rest } = config;
    const body: AssistantConfigUpdate = { ...rest };
    (Object.keys(secrets) as SecretKey[]).forEach((key) => {
      if (secrets[key]) body[key] = secrets[key];
    });
    try {
      if (applyResponse(await api.setAssistantConfig(body))) {
        setSecrets(emptySecrets);
        message.success(t('settings.assistant.saved'));
      }
    } finally {
      setIsSaving(false);
    }
  }

  async function clearSecret(flag: 'clearGeminiApiKey' | 'clearOrApiKey' | 'clearProxyPass') {
    applyResponse(await api.setAssistantConfig({ [flag]: true }));
  }

  async function upload(file: File) {
    const rsp = await api.uploadAttachment(file);
    if (rsp.code !== 0) message.error(rsp.msg);
    loadAttachments();
  }

  async function remove(name: string) {
    const rsp = await api.deleteAttachment(name);
    if (rsp.code !== 0) message.error(rsp.msg);
    loadAttachments();
  }

  if (!config) return null;

  const secretInput = (key: SecretKey, has: boolean, flag: Parameters<typeof clearSecret>[0]) => (
    <>
      <Input.Password
        className="w-[260px]"
        value={secrets[key]}
        placeholder={has ? t('settings.assistant.secretSet') : ''}
        onChange={(e) => setSecrets((s) => ({ ...s, [key]: e.target.value }))}
      />
      {has && <Button onClick={() => clearSecret(flag)}>{t('settings.assistant.clear')}</Button>}
    </>
  );

  return (
    <>
      <Row label={t('settings.assistant.enabled')} desc={t('settings.assistant.enabledDesc')}>
        <Switch checked={config.enabled} onChange={(v) => update('enabled', v)} />
      </Row>
      <Row label={t('settings.assistant.ui')} desc={t('settings.assistant.uiDesc')}>
        <Switch checked={config.ui} onChange={(v) => update('ui', v)} />
      </Row>

      <Divider />
      <Row label={t('settings.assistant.provider')}>
        <Select
          className="w-[260px]"
          value={config.provider}
          onChange={(v) => update('provider', v)}
          options={[
            { value: 'openrouter', label: 'OpenRouter' },
            { value: 'gemini', label: 'Gemini' }
          ]}
        />
      </Row>
      <div style={{ opacity: config.provider === 'gemini' ? 1 : 0.5 }}>
        <div className="pt-2 font-medium">Gemini</div>
        <Row label={t('settings.assistant.baseUrl')}>
          <Input
            className="w-[260px]"
            value={config.geminiBaseUrl}
            onChange={(e) => update('geminiBaseUrl', e.target.value)}
          />
        </Row>
        <Row label={t('settings.assistant.model')}>
          <Input
            className="w-[260px]"
            value={config.geminiModel}
            onChange={(e) => update('geminiModel', e.target.value)}
          />
        </Row>
        <Row label={t('settings.assistant.apiKey')}>
          {secretInput('geminiApiKey', config.hasGeminiApiKey, 'clearGeminiApiKey')}
        </Row>
        <Row label={t('settings.assistant.thinking')}>
          <Switch checked={config.geminiThinking} onChange={(v) => update('geminiThinking', v)} />
        </Row>
        <Row label={t('settings.assistant.thinkingBudget')}>
          <InputNumber
            min={0}
            step={512}
            value={config.thinkingBudget}
            onChange={(v) => update('thinkingBudget', v ?? 512)}
          />
        </Row>
      </div>
      <div style={{ opacity: config.provider === 'openrouter' ? 1 : 0.5 }}>
        <div className="pt-2 font-medium">OpenRouter</div>
        <Row label={t('settings.assistant.baseUrl')}>
          <Input
            className="w-[260px]"
            value={config.orBaseUrl}
            onChange={(e) => update('orBaseUrl', e.target.value)}
          />
        </Row>
        <Row label={t('settings.assistant.model')}>
          <Input
            className="w-[260px]"
            value={config.orModel}
            onChange={(e) => update('orModel', e.target.value)}
          />
        </Row>
        <Row label={t('settings.assistant.apiKey')}>
          {secretInput('orApiKey', config.hasOrApiKey, 'clearOrApiKey')}
        </Row>
        <Row label={t('settings.assistant.reasoning')}>
          <Switch checked={config.orReasoning} onChange={(v) => update('orReasoning', v)} />
        </Row>
        <Row label={t('settings.assistant.reasoningEffort')}>
          <Select
            className="w-[140px]"
            value={config.orReasoningEffort}
            onChange={(v) => update('orReasoningEffort', v)}
            options={['low', 'medium', 'high'].map((v) => ({ value: v, label: v }))}
          />
        </Row>
      </div>

      <Divider />
      <div className="font-medium">{t('settings.assistant.proxy')}</div>
      <Row label={t('settings.assistant.proxyUrl')}>
        <Input
          className="w-[260px]"
          value={config.proxyUrl}
          onChange={(e) => update('proxyUrl', e.target.value)}
        />
      </Row>
      <Row label={t('settings.assistant.proxyPass')}>
        {secretInput('proxyPass', config.hasProxyPass, 'clearProxyPass')}
      </Row>

      <Divider />
      <div className="font-medium">{t('settings.assistant.behaviour')}</div>
      {(['copyClipboard', 'answerSel', 'contextSel', 'frqSel', 'quadClickMCQ'] as const).map(
        (key) => (
          <Row key={key} label={t(`settings.assistant.${key}`)}>
            <Switch checked={config[key]} onChange={(v) => update(key, v)} />
          </Row>
        )
      )}

      <div className="flex justify-end pt-4">
        <Button type="primary" loading={isSaving} onClick={save}>
          {t('settings.assistant.save')}
        </Button>
      </div>

      <Divider />
      <Row
        label={t('settings.assistant.attachments')}
        desc={t('settings.assistant.attachmentsDesc')}
      >
        <Upload
          showUploadList={false}
          customRequest={({ file, onSuccess }) => {
            upload(file as File).then(() => onSuccess?.(null));
          }}
        >
          <Button icon={<UploadIcon size={14} />}>{t('settings.assistant.upload')}</Button>
        </Upload>
      </Row>
      {attachments.map((a) => (
        <div key={a.name} className="flex items-center justify-between py-1 text-sm">
          <span>
            {a.name} <span className="text-neutral-500">({Math.ceil(a.size / 1024)} KB)</span>
          </span>
          <Button type="text" icon={<Trash2Icon size={14} />} onClick={() => remove(a.name)} />
        </div>
      ))}

      <Divider />
      <div className="font-medium">{t('settings.assistant.hotkeys')}</div>
      <div className="pt-1 text-sm text-neutral-400">{t('settings.assistant.hotkeyList')}</div>
    </>
  );
};
