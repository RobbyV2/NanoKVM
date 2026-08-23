import type { Binding, SourcesEvent, SourceSink, SourcesSnapshot } from '@/api/sources.ts';

export const emptySnapshot: SourcesSnapshot = { sinks: [], sources: [], bindings: [] };

function replaceBinding(sinks: SourceSink[], binding: Binding | null, sinkID: string) {
  return sinks.map((sink) => (sink.id === sinkID ? { ...sink, binding } : sink));
}

function output(sink: SourceSink): SourceSink['output'] {
  if (!sink.demand.streaming) return 'idle';
  if (sink.binding?.state === 'streaming') return 'source';
  return sink.kind === 'camera' ? 'black' : 'silence';
}

function refreshOutputs(sinks: SourceSink[]) {
  return sinks.map((sink) => ({ ...sink, output: output(sink) }));
}

export function reduceSources(snapshot: SourcesSnapshot, event: SourcesEvent): SourcesSnapshot {
  if (event.type === 'snapshot' && event.snapshot) return event.snapshot;
  if (event.type === 'sinks_changed' && event.sinks) {
    return { ...snapshot, sinks: event.sinks, bindings: event.sinks.flatMap(bindingOf) };
  }
  if (event.type === 'source_added' && event.source) {
    return {
      ...snapshot,
      sources: [
        ...snapshot.sources.filter((source) => source.id !== event.source?.id),
        event.source
      ]
    };
  }
  if (event.type === 'source_removed' && event.source) {
    return {
      ...snapshot,
      sources: snapshot.sources.filter((source) => source.id !== event.source?.id)
    };
  }
  if (event.type === 'binding_added' && event.binding) {
    const bindings = [
      ...snapshot.bindings.filter((binding) => binding.sink_id !== event.binding?.sink_id),
      event.binding
    ];
    return {
      ...snapshot,
      bindings,
      sinks: refreshOutputs(replaceBinding(snapshot.sinks, event.binding, event.binding.sink_id))
    };
  }
  if (event.type === 'binding_state' && event.binding) {
    const bindings = snapshot.bindings.map((binding) =>
      binding.sink_id === event.binding?.sink_id ? event.binding : binding
    );
    return {
      ...snapshot,
      bindings,
      sinks: refreshOutputs(replaceBinding(snapshot.sinks, event.binding, event.binding.sink_id))
    };
  }
  if (event.type === 'binding_removed' && event.binding) {
    return {
      ...snapshot,
      bindings: snapshot.bindings.filter((binding) => binding.sink_id !== event.binding?.sink_id),
      sinks: refreshOutputs(replaceBinding(snapshot.sinks, null, event.binding.sink_id))
    };
  }
  if (event.type === 'demand' && event.sink_id && event.demand) {
    const sinks = snapshot.sinks.map((sink) =>
      sink.id === event.sink_id ? { ...sink, demand: event.demand! } : sink
    );
    return { ...snapshot, sinks: refreshOutputs(sinks) };
  }
  return snapshot;
}

function bindingOf(sink: SourceSink) {
  return sink.binding ? [sink.binding] : [];
}
