import { useEffect, useRef, useState } from 'react';
import { Button, Input, message, Popconfirm, Tag } from 'antd';
import { PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/assistant.ts';
import type { PromptEntry, PromptSet } from '@/api/assistant.ts';

// Local ids keep React keys stable while names and keys are being edited.
type DraftField = { id: number; key: string; value: string };
type DraftEntry = { id: number; name: string; fields: DraftField[] };

type PromptsResponse = { code: number; msg: string; data: PromptSet };

export const PromptsSection = () => {
  const { t } = useTranslation();
  const nextId = useRef(0);
  const [entries, setEntries] = useState<DraftEntry[] | null>(null);
  const [isDefault, setIsDefault] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  const newId = () => ++nextId.current;

  function toDraft(list: PromptEntry[]): DraftEntry[] {
    return list.map((e) => ({
      id: newId(),
      name: e.name,
      fields: e.fields.map((f) => ({ id: newId(), key: f.key, value: f.value }))
    }));
  }

  function apply(rsp: PromptsResponse) {
    if (rsp.code !== 0) {
      message.error(rsp.msg);
      return false;
    }
    setEntries(toDraft(rsp.data.entries));
    setIsDefault(rsp.data.isDefault);
    return true;
  }

  useEffect(() => {
    api.getPrompts().then((rsp) => apply(rsp));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function updateEntry(id: number, fn: (e: DraftEntry) => DraftEntry) {
    setEntries((list) => list && list.map((e) => (e.id === id ? fn(e) : e)));
  }

  function updateField(entryId: number, fieldId: number, patch: Partial<DraftField>) {
    updateEntry(entryId, (e) => ({
      ...e,
      fields: e.fields.map((f) => (f.id === fieldId ? { ...f, ...patch } : f))
    }));
  }

  function addEntry() {
    setEntries((list) => [
      ...(list ?? []),
      { id: newId(), name: '', fields: [{ id: newId(), key: 'prompt', value: '' }] }
    ]);
  }

  function removeEntry(id: number) {
    setEntries((list) => list && list.filter((e) => e.id !== id));
  }

  function addField(entryId: number) {
    updateEntry(entryId, (e) => ({
      ...e,
      fields: [...e.fields, { id: newId(), key: '', value: '' }]
    }));
  }

  function removeField(entryId: number, fieldId: number) {
    updateEntry(entryId, (e) => ({ ...e, fields: e.fields.filter((f) => f.id !== fieldId) }));
  }

  async function save() {
    if (!entries) return;
    setIsSaving(true);
    try {
      const body: PromptEntry[] = entries.map((e) => ({
        name: e.name,
        fields: e.fields.map((f) => ({ key: f.key, value: f.value }))
      }));
      if (apply(await api.savePrompts(body))) {
        message.success(t('settings.assistant.promptsSaved'));
      }
    } finally {
      setIsSaving(false);
    }
  }

  async function reset() {
    if (apply(await api.resetPrompts())) {
      message.success(t('settings.assistant.promptsResetDone'));
    }
  }

  if (!entries) return null;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-4">
        <div className="flex flex-col">
          <span className="font-medium">
            {t('settings.assistant.prompts')}{' '}
            <Tag color={isDefault ? 'default' : 'blue'}>
              {isDefault
                ? t('settings.assistant.promptsDefault')
                : t('settings.assistant.promptsCustomized')}
            </Tag>
          </span>
          <span className="text-xs text-neutral-500">{t('settings.assistant.promptsDesc')}</span>
        </div>
      </div>

      {entries.map((entry) => (
        <div key={entry.id} className="flex flex-col gap-2 rounded border border-neutral-700 p-3">
          <div className="flex items-center gap-2">
            <Input
              className="w-[260px]"
              value={entry.name}
              placeholder={t('settings.assistant.promptsEntryName')}
              onChange={(e) => updateEntry(entry.id, (x) => ({ ...x, name: e.target.value }))}
            />
            <div className="flex-1" />
            <Button danger icon={<Trash2Icon size={14} />} onClick={() => removeEntry(entry.id)}>
              {t('settings.assistant.promptsRemoveEntry')}
            </Button>
          </div>

          {entry.fields.map((field) => (
            <div key={field.id} className="flex flex-col gap-1 pl-2">
              <div className="flex items-center gap-2">
                <Input
                  className="w-[180px]"
                  value={field.key}
                  placeholder={t('settings.assistant.promptsFieldKey')}
                  onChange={(e) => updateField(entry.id, field.id, { key: e.target.value })}
                />
                <Button
                  type="text"
                  icon={<Trash2Icon size={14} />}
                  onClick={() => removeField(entry.id, field.id)}
                >
                  {t('settings.assistant.promptsRemoveField')}
                </Button>
              </div>
              <Input.TextArea
                className="w-full font-mono"
                style={{ resize: 'vertical' }}
                autoSize={{ minRows: 8, maxRows: 30 }}
                value={field.value}
                onChange={(e) => updateField(entry.id, field.id, { value: e.target.value })}
              />
            </div>
          ))}

          <div>
            <Button icon={<PlusIcon size={14} />} onClick={() => addField(entry.id)}>
              {t('settings.assistant.promptsAddField')}
            </Button>
          </div>
        </div>
      ))}

      <div>
        <Button icon={<PlusIcon size={14} />} onClick={addEntry}>
          {t('settings.assistant.promptsAddEntry')}
        </Button>
      </div>

      <div className="flex justify-end gap-2">
        <Popconfirm
          placement="top"
          title={t('settings.assistant.promptsResetConfirm')}
          okText={t('settings.assistant.promptsResetOk')}
          cancelText={t('settings.assistant.promptsResetCancel')}
          onConfirm={reset}
        >
          <Button>{t('settings.assistant.promptsReset')}</Button>
        </Popconfirm>
        <Button type="primary" loading={isSaving} onClick={save}>
          {t('settings.assistant.promptsSave')}
        </Button>
      </div>
    </div>
  );
};
