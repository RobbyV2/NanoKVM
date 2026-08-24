const it = {
  translation: {
    head: {
      desktop: 'Desktop Remoto',
      login: 'Accesso',
      changePassword: 'Cambia Password',
      terminal: 'Terminale',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Accesso',
      placeholderUsername: 'Inserisci il nome utente',
      placeholderPassword: 'Inserisci la password',
      placeholderCurrentPassword: 'Password attuale',
      placeholderPassword2: 'Inserisci nuovamente la password',
      noEmptyUsername: 'Il nome utente non può essere vuoto',
      noEmptyPassword: 'La password non può essere vuota',
      passwordLength: 'La password deve contenere tra 8 e 72 caratteri',
      noAccount:
        'Impossibile ottenere informazioni utente, aggiorna la pagina o reimposta la password',
      invalidUser: 'Nome utente o password non validi',
      locked: 'Troppi accessi, riprova più tardi',
      globalLocked: 'Sistema sotto protezione, riprova più tardi',
      error: 'Errore imprevisto',
      invalidCurrentPassword: 'La password attuale non è corretta',
      changePassword: 'Cambia Password',
      changePasswordDesc:
        'Per la sicurezza del tuo dispositivo, modifica la password di accesso web.',
      differentPassword: 'Le password non corrispondono',
      illegalUsername: 'Il nome utente contiene caratteri non validi',
      illegalPassword: 'La password contiene caratteri non validi',
      forgetPassword: 'Hai dimenticato la password',
      ok: 'Ok',
      cancel: 'Annulla',
      loginButtonText: 'Accedi',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'Per i passaggi dettagliati, consulta questo documento:',
        reset3: 'Account web predefinito:',
        reset4: 'Account SSH predefinito:',
        change1: 'Tieni presente che questa azione modificherà le seguenti password:',
        change2: 'Password di accesso web',
        change3: 'Password root di sistema (password di accesso SSH)',
        change4: 'Per reimpostare le password, tieni premuto il pulsante BOOT sul NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Configura il Wi-Fi per NanoKVM',
      success: 'Please check the network status of NanoKVM and visit the new IP address.',
      failed: 'Operazione non riuscita, riprova.',
      invalidMode:
        'La modalità corrente non supporta la configurazione di rete. Vai al tuo dispositivo e abilita la modalità di configurazione Wi-Fi.',
      confirmBtn: 'Ok',
      finishBtn: 'Completato',
      ap: {
        authTitle: 'Autenticazione richiesta',
        authDescription: 'Inserisci la password AP per continuare',
        authFailed: 'Password AP non valida',
        passPlaceholder: 'AP password',
        verifyBtn: 'Verifica'
      }
    },
    screen: {
      scale: 'Scala',
      title: 'Schermo',
      video: 'Modalità video',
      videoDirectTips:
        'Abilita HTTPS in "Impostazioni > Dispositivo" per utilizzare questa modalità',
      resolution: 'Risoluzione',
      auto: 'Automatico',
      autoTips:
        'Potrebbero verificarsi tearing dello schermo o spostamento del mouse a risoluzioni specifiche. Considera di regolare la risoluzione del dispositivo remoto o disabilitare la modalità automatica.',
      fps: 'FPS',
      customizeFps: 'Personalizza',
      quality: 'Qualità',
      qualityLossless: 'Senza perdita',
      qualityHigh: 'Alto',
      qualityMedium: 'Medio',
      qualityLow: 'Basso',
      frameDetect: 'Rilevamento Frame',
      frameDetectTip:
        'Calcola la differenza tra i frame. Interrompe la trasmissione del flusso video quando non vengono rilevate modifiche sullo schermo del dispositivo remoto.',
      resetHdmi: 'Reimposta HDMI',
      mixedH264: {
        title: 'Conflitto del flusso H.264',
        description:
          'I flussi H.264 Direct e H.264 WebRTC sono utilizzati contemporaneamente. Ciò può causare tearing dello schermo o video danneggiato. Utilizzare una sola modalità H.264.'
      },
      webrtcConnectionFailed: {
        title: 'Connessione WebRTC non riuscita',
        description: 'Controlla la connessione di rete o cambia la modalità video.'
      },
      captureStatus: {
        hdmiError: 'Errore schermata HDMI',
        unsupportedResolution: 'La risoluzione attuale non è supportata',
        retrieving: 'Acquisizione schermata...',
        changingResolution: 'Cambio risoluzione...',
        updateFailed: 'Lo schermo non può aggiornarsi al momento',
        videoError: 'Errore di visualizzazione video',
        noHdmi: 'Nessun segnale HDMI rilevato',
        unavailable: 'Lo schermo non può essere visualizzato al momento'
      }
    },
    keyboard: {
      title: 'Tastiera',
      paste: 'Incolla',
      tips: 'Sono supportati solo lettere e simboli standard della tastiera',
      placeholder: 'Inserisci testo',
      submit: 'Invia',
      virtual: 'Tastiera',
      readClipboard: 'Leggi dagli Appunti',
      clipboardPermissionDenied:
        "Autorizzazione Appunti negata. Consenti l'accesso agli appunti nel tuo browser.",
      clipboardReadError: 'Impossibile leggere gli appunti',
      dropdownEnglish: 'Inglese',
      dropdownGerman: 'Tedesco',
      dropdownFrench: 'Francese',
      dropdownRussian: 'Russo',
      shortcut: {
        title: 'Scorciatoie',
        custom: 'Personalizzato',
        capture: 'Fai clic qui per acquisire il collegamento',
        clear: 'Cancella',
        save: 'Salva',
        captureTips:
          'La cattura dei tasti di sistema (come il tasto Windows) richiede l’autorizzazione a schermo intero.',
        enterFullScreen: 'Attiva/disattiva la modalità a schermo intero.'
      },
      leaderKey: {
        title: 'Tasto Leader',
        desc: "Ignora le restrizioni del browser e invia collegamenti di sistema direttamente all'host remoto.",
        howToUse: 'Come usare',
        simultaneous: {
          title: 'Modalità simultanea',
          desc1: 'Tieni premuto il tasto Leader, quindi premi la scorciatoia.',
          desc2: 'Intuitivo, ma potrebbe entrare in conflitto con le scorciatoie di sistema.'
        },
        sequential: {
          title: 'Modalità sequenziale',
          desc1:
            'Premi il tasto Leader → premi la scorciatoia in sequenza → premi di nuovo il tasto Leader.',
          desc2: 'Richiede più passaggi, ma evita completamente i conflitti di sistema.'
        },
        enable: 'Abilita tasto Leader',
        tip: 'Quando assegnato come tasto Leader, questo tasto funziona solo come attivatore di scorciatoie e perde il comportamento predefinito.',
        placeholder: 'Premi il tasto Leader',
        shiftRight: 'Shift destro',
        ctrlRight: 'Ctrl destro',
        metaRight: 'Win destro',
        submit: 'Invia',
        recorder: {
          rec: 'REC',
          activate: 'Attiva i tasti',
          input: 'Premi la scorciatoia...'
        }
      }
    },
    mouse: {
      title: 'Mouse',
      cursor: 'Stile cursore',
      default: 'Cursore predefinito',
      pointer: 'Cursore a puntatore',
      cell: 'Cursore a cella',
      text: 'Cursore testo',
      grab: 'Cursore di presa',
      hide: 'Nascondi cursore',
      mode: 'Modalità mouse',
      absolute: 'Modalità assoluta',
      relative: 'Modalità relativa',
      direction: 'Direzione della rotellina',
      scrollUp: "Scorri verso l'alto",
      scrollDown: 'Scorri verso il basso',
      speed: 'Velocità della rotellina',
      fast: 'Veloce',
      slow: 'Lento',
      requestPointer:
        'Usando la modalità relativa. Clicca sul desktop per ottenere il puntatore del mouse.',
      resetHid: 'Reimposta HID',
      hidOnly: {
        title: 'Modalità solo HID',
        desc: 'Se il mouse e la tastiera smettono di rispondere e il ripristino di HID non aiuta, potrebbe trattarsi di un problema di compatibilità tra NanoKVM e il dispositivo. Prova ad abilitare la modalità HID-Only per una migliore compatibilità.',
        tip1: "L'abilitazione della modalità HID-Solo smonterà il disco U virtuale e la rete virtuale",
        tip2: "Nella modalità HID-Only, il montaggio dell'immagine è disabilitato",
        tip3: 'NanoKVM si riavvierà automaticamente dopo aver cambiato modalità',
        enable: 'Abilita la modalità HID-Solo',
        disable: 'Disabilita la modalità HID-Solo'
      }
    },
    image: {
      title: 'Immagini',
      loading: 'Caricamento...',
      empty: 'Nessun risultato',
      mountMode: 'Modalità di montaggio',
      mountFailed: 'Montaggio immagine fallito',
      mountDesc:
        "In alcuni sistemi, è necessario espellere il disco virtuale sull'host remoto prima di montare l'immagine.",
      unmountFailed: 'Smontaggio non riuscito',
      unmountDesc:
        "Su alcuni sistemi, è necessario espellere manualmente l'host remoto prima di smontare l'immagine.",
      refresh: "Aggiorna l'elenco delle immagini",
      attention: 'Attenzione',
      deleteConfirm: 'Sei sicuro di voler eliminare questa immagine?',
      okBtn: 'Sì',
      cancelBtn: 'No',
      tips: {
        title: 'Come caricare',
        usb1: 'Collega il NanoKVM al tuo computer tramite USB.',
        usb2: 'Assicurati che la Virtual Disk sia montata (Impostazioni - Virtual Disk).',
        usb3: 'Apri il disk virtuale sul tuo computer e copia il file immagine nella directory principale del disk.',
        scp1: 'Assicurati che il NanoKVM e il tuo computer siano sulla stessa rete locale.',
        scp2: 'Apri un terminale sul tuo computer e usa il comando SCP per caricare il file immagine nella directory /data del NanoKVM.',
        scp3: 'Esempio: scp il-tuo-percorso-immagine root@il-tuo-ip-nanokvm:/data',
        tfCard: 'Scheda TF',
        tf1: 'Questo metodo è supportato su sistemi Linux',
        tf2: 'Recupera la scheda TF dal NanoKVM (per la versione FULL, smonta prima il case).',
        tf3: 'Inserisci la scheda TF in un lettore di schede e collegala al tuo computer.',
        tf4: 'Copia il file immagine nella directory /data sulla scheda TF.',
        tf5: 'Inserisci la scheda TF nel NanoKVM.'
      }
    },
    script: {
      title: 'Script',
      upload: 'Carica',
      run: 'Esegui',
      runBackground: 'Esegui in Background',
      runFailed: 'Esecuzione fallita',
      attention: 'Attenzione',
      delDesc: 'Sei sicuro di voler eliminare questo file?',
      confirm: 'Sì',
      cancel: 'No',
      delete: 'Elimina',
      close: 'Chiudi'
    },
    terminal: {
      title: 'Terminale',
      nanokvm: 'Terminale NanoKVM',
      serial: 'Terminale Porta Seriale',
      serialPort: 'Porta Seriale',
      serialPortPlaceholder: 'Inserisci la porta seriale',
      baudrate: 'Baud rate',
      parity: 'Parità',
      parityNone: 'Nessuno',
      parityEven: 'Pari',
      parityOdd: 'Dispari',
      flowControl: 'Controllo del flusso',
      flowControlNone: 'Nessuno',
      flowControlSoft: 'Software',
      flowControlHard: 'Hardware',
      dataBits: 'Bit di dati',
      stopBits: 'Bit di stop',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Invio comando...',
      sent: 'Comando inviato',
      input: 'Inserisci il MAC',
      ok: 'Ok'
    },
    download: {
      title: 'Scaricatore di immagini',
      input: "Inserisci un'immagine remota URL",
      ok: 'Ok',
      disabled: "La partizione /data è RO, quindi non possiamo scaricare l'immagine",
      uploadbox: 'Rilascia il file qui o fai clic per selezionarlo',
      inputfile: 'Inserisci il file immagine',
      NoISO: 'Nessuna ISO',
      sha256: 'SHA-256 (facoltativo)',
      sha256Placeholder: 'Inserisci un checksum SHA-256 di 64 caratteri',
      invalidSHA256: 'SHA-256 deve essere una stringa esadecimale di 64 caratteri',
      failed: 'Download non riuscito',
      success: 'Download riuscito',
      checksumFailed: 'Download non riuscito: verifica SHA-256 non riuscita',
      cancel: 'Annulla',
      cancelFailed: 'Impossibile annullare il download'
    },
    power: {
      title: 'Accensione',
      showConfirm: 'Conferma',
      showConfirmTip: 'Le operazioni di alimentazione richiedono una conferma aggiuntiva',
      reset: 'Reimposta',
      power: 'Accensione',
      powerShort: 'Accensione (clic breve)',
      powerLong: 'Accensione (clic lungo)',
      resetConfirm: "Procedere con l'operazione di ripristino?",
      powerConfirm: "Procedere con l'operazione di accensione?",
      okBtn: 'Sì',
      cancelBtn: 'No'
    },
    devices: {
      title: 'Dispositivi',
      stale: 'Lo stato in tempo reale dei dispositivi non è disponibile. Riconnessione in corso.',
      empty:
        'Nessuno slot per fotocamera o microfono è configurato. Aggiungine uno in Impostazioni, Dispositivo.',
      available: 'Disponibile',
      waiting: 'L’host attende una sorgente',
      hostOpen: 'Host aperto',
      hostIdle: 'Host inattivo',
      sending: 'Invio da questo browser',
      black: 'Video nero',
      silence: 'Silenzio digitale',
      resuming: 'In attesa di riprendere',
      stop: 'Interrompi la condivisione',
      disconnect: 'Disconnetti',
      takeover: 'Prendi il controllo',
      refused: 'In uso da {{owner}} tramite {{source}}',
      connectedSources_one: '{{count}} sorgente collegata',
      connectedSources_other: '{{count}} sorgenti collegate',
      connectedSources: '{{count}} sorgenti collegate',
      connection: {
        connecting: 'Connessione',
        connected: 'In diretta',
        disconnected: 'Riconnessione'
      },
      share: {
        camera: 'Condividi la fotocamera',
        microphone: 'Condividi il microfono',
        usbDevice: 'Condividi USB'
      },
      permission: {
        denied: 'Bloccato nelle impostazioni del sito del browser',
        prompt: 'Il browser chiederà l’autorizzazione',
        insecure:
          'Questa pagina non è servita via HTTPS, quindi il browser blocca questo dispositivo. Abilita HTTPS in Impostazioni, Rete.'
      },
      capture: {
        unsupported: 'Questo browser non può acquisire audio o video',
        camera: 'Questo browser non può codificare i fotogrammi della fotocamera',
        microphone: 'Questo browser non può elaborare l’audio del microfono'
      },
      mic: {
        mute: 'Disattiva audio',
        unmute: 'Riattiva audio'
      },
      revoked: {
        released: 'La condivisione è stata interrotta',
        lease_expired: 'Il lease è scaduto prima che questo browser tornasse',
        admin_disconnect: 'Un amministratore ha disconnesso tutte le sorgenti',
        slot_removed: 'Lo slot è stato rimosso',
        slot_changed: 'Lo slot è stato riconfigurato',
        taken_over: 'Un amministratore ha preso questo slot'
      },
      usb: {
        surrendered: 'Il passthrough USB tiene tastiera e mouse',
        surrenderedDesc:
          'L’host remoto vede il dispositivo importato al posto di tastiera, mouse e supporti virtuali del NanoKVM. Tornano quando la sessione si ferma.',
        unsupported: 'WebUSB richiede un browser Chromium',
        insecure:
          'Questa pagina non è servita via HTTPS, quindi il browser blocca WebUSB. Abilita HTTPS in Impostazioni, Rete.',
        session: 'Inoltro di {{device}} ({{mode}})',
        idle: 'Nessuna sessione di passthrough',
        mode: {
          hybrid: 'ibrido',
          exact: 'esatto'
        }
      }
    },
    settings: {
      title: 'Impostazioni',
      display: {
        title: 'Schermo',
        loading: 'Caricamento...',
        active: 'EDID attivo',
        activeUnknown:
          "NanoKVM non ha scritto alcun EDID dall'avvio, quindi l'identità vista dall'host è sconosciuta.",
        appliedAt: 'Applicato il {{time}}',
        download: 'Scarica',
        downloadBackup: 'Scarica il precedente',
        preset: 'Preset monitor',
        presetPlaceholder: 'Seleziona un monitor',
        upload: 'Carica',
        selected: 'EDID selezionato',
        errors: 'Errori',
        warnings: 'Avvisi',
        info: 'Informazioni',
        unknownMonitor: 'Monitor sconosciuto',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Audio',
        audioNo: 'Nessun audio',
        extensionBlocks: 'Blocchi di estensione: {{blocks}}',
        apply: 'Applica',
        applyTitle: 'Applicare questo EDID?',
        before: 'Attuale',
        after: 'Nuovo',
        hdmiNotice:
          "L'acquisizione video si interrompe durante la scrittura dell'EDID e riprende da sola al termine.",
        powerCycleNotice:
          'Questo dispositivo deve essere scollegato fisicamente dalla corrente e ricollegato prima che il nuovo EDID abbia effetto.',
        powerCycleUnverified:
          'La scrittura non è stata verificata, quindi il chip video mantiene ciò che contiene adesso finché questo dispositivo non viene fisicamente scollegato dall’alimentazione e ricollegato.',
        applied: 'EDID applicato e verificato.',
        applyFailed: "Applicazione dell'EDID non riuscita.",
        busy: 'Il chip video era occupato. Riprova.',
        unsupported: "Questo dispositivo non supporta la modifica dell'EDID.",
        toolMissing: 'Lo strumento EDID non è presente in questo firmware.',
        noAudio:
          "Questo EDID non dichiara audio, quindi l'host potrebbe smettere di inviare il suono.",
        oldVersion: 'Questo EDID usa una versione precedente alla 1.4.',
        interlaced: 'La risoluzione preferita è interlacciata.',
        tooLarge:
          'La risoluzione preferita supera 1920x1080 a 60 Hz, oltre quanto NanoKVM può acquisire.',
        recovery: 'Ripristino',
        recoveryNeeded:
          "L'ultima scrittura non è stata verificata, quindi l'area EDID del chip video si trova in uno stato sconosciuto. Ripristina l'EDID di fabbrica per tornare a uno stato noto.",
        recoveryDesc:
          "Riscrive un EDID noto sul chip video quando quello applicato ha lasciato l'host senza immagine.",
        restoreFactory: "Ripristina l'EDID di fabbrica",
        restoreBackup: 'Ripristina EDID precedente',
        restoreTitle: 'Ripristinare questo EDID?',
        restoreFactoryTarget: "L'EDID di fabbrica fornito con NanoKVM.",
        restoreBackupTarget: 'Il backup più recente, applicato il {{time}}.',
        restoreNotice:
          "Un ripristino viene scritto come un'applicazione e comporta le stesse conseguenze.",
        restored: 'EDID ripristinato e verificato.',
        restoreFailed: "Ripristino dell'EDID non riuscito.",
        mismatchTitle: 'Scritto e riletto',
        mismatchWritten: 'Scritto',
        mismatchRead: 'Riletto',
        restoreOkBtn: 'Ripristina',
        hardware: 'Hardware rilevato: {{hardware}}',
        hardwareUnknown: 'Sconosciuto',
        confirmWord: 'APPLICA',
        confirmPrompt: 'Digita {{word}} per abilitare il pulsante di applicazione.',
        okBtn: 'Applica',
        cancelBtn: 'Annulla'
      },
      presentation: {
        title: 'Presentazione USB',
        loading: 'Caricamento...',
        current: 'Presentazione USB attuale',
        noProfile: 'Nessun profilo applicato',
        linked: 'Funzioni collegate',
        hostState: 'USB dell’host',
        hostUnbound: 'Controller non associato',
        hdmiState: 'Ingresso HDMI',
        hdmiSignal: 'Segnale presente',
        hdmiUnreported: 'Ancora nessun rapporto di acquisizione',
        endpoints: 'Endpoint',
        fifos: 'Slot FIFO',
        pending: 'Modifiche in sospeso',
        pendingEdits: 'Modifiche all’identità non salvate',
        pendingProfile: '{{profile}} è selezionato ma non applicato',
        pendingNone: 'Nessuna',
        lastApply: 'Ultima applicazione',
        applyFailed: 'Fallita su {{profile}} il {{time}}',
        applyClean: 'Nessun errore registrato',
        lastKnownGood: 'Ultimo stato valido noto',
        rollbackTarget: 'Destinazione del ripristino',
        rollbackNone: 'Nessuna',
        powerCyclePending:
          'Il controller è stato sottratto all’host. Spegni e riaccendi il computer collegato per riavere il dispositivo.',
        rollback: 'Ripristina',
        rollbackTitle: 'Ripristinare {{profile}}?',
        rollbackDesc: 'Il gadget viene rienumerato; le funzioni USB cadono per un istante.',
        profile: 'Profilo USB',
        builtIn: 'integrato',
        descriptors: 'descrittori',
        imported: 'importato',
        clone: 'Clona',
        cloneTitle: 'Clona questo profilo',
        cloneToEdit:
          'I profili integrati restano in sola lettura. Clona questo profilo per modificarne l’identità.',
        profileName: 'Nome del profilo',
        profileNameHint: 'Lettere minuscole, cifre, punti, trattini bassi e trattini.',
        import: 'Importa pacchetto',
        export: 'Esporta pacchetto',
        delete: 'Elimina',
        deleteTitle: 'Eliminare questo profilo?',
        deleteDesc: 'Elimina {{profile}} dal NanoKVM.',
        identity: 'Identità USB',
        preset: 'Identità preimpostata',
        presetPlaceholder: 'Copia l’identità da un dispositivo noto',
        presetHint:
          'Un preset riempie il Vendor ID, il Product ID e i due campi del nome. Non porta con sé alcun descrittore.',
        presetSource: 'Identità così come registrata in {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Questo Vendor ID appartiene a un altro produttore',
        productId: 'Product ID',
        bcdUSB: 'Versione USB',
        bcdDevice: 'Versione del dispositivo',
        manufacturer: 'Produttore',
        product: 'Prodotto',
        serial: 'Numero di serie',
        configuration: 'Stringa di configurazione',
        hidLayout: 'Dispositivi HID',
        hidRoleKeyboard: 'Tastiera',
        hidRoleRelative: 'Mouse (relativo)',
        hidRoleAbsolute: 'Puntatore (assoluto)',
        hidOff: 'Non presente',
        hidInterface: 'Interfaccia {{index}}',
        hidBootKeyboardShared:
          'La tastiera condivide un’interfaccia, quindi non offre più un report in protocollo boot. Alcuni BIOS e UEFI non la vedranno.',
        functions: 'Funzioni',
        descriptorAssets: 'Descrittori memorizzati: {{count}}',
        endpointUse:
          'IN {{inUse}} in uso, {{inFree}} liberi; OUT {{outUse}} in uso, {{outFree}} liberi',
        apply: 'Applica',
        applyTitle: 'Applicare questo profilo USB?',
        applyDesc: 'Il NanoKVM presenterà {{profile}} al computer collegato.',
        reconnect:
          'Tastiera, mouse e le altre funzioni USB si disconnettono per un istante mentre il gadget viene ricollegato.',
        applyLinks: 'Collega: {{functions}}',
        applyRemoves: 'Rimuove: {{functions}}',
        applyNoHid:
          'Dopo questa applicazione non resta alcuna funzione HID. Tastiera e mouse smetteranno di funzionare.',
        applyRollback: 'Un’applicazione fallita torna a {{profile}}.',
        recoveryPowerCycle:
          'Nessun HID sopravvive a questa applicazione, quindi un host che smette di rispondere si può recuperare solo togliendo e ridando alimentazione.',
        recoveryReboot:
          'Un’interfaccia sparisce dal dispositivo composito; l’host potrebbe aver bisogno di un riavvio per riassociare il resto.',
        recoveryHdmiReset:
          'Una funzione video viene ricostruita, perciò la catena di acquisizione che sta dietro si resetta.',
        recoveryReconnect:
          'L’host rienumera il dispositivo; le funzioni USB cadono per un istante.',
        cancel: 'Annulla',
        noFunctions: 'Nessuna funzione collegata',
        loadFailed: 'Caricamento dei profili di presentazione non riuscito',
        operationFailed: 'Operazione di presentazione non riuscita'
      },
      passthrough: {
        title: 'Passthrough USB',
        loading: 'Caricamento...',
        mode: 'Modalità',
        hybrid: 'Ibrida',
        exact: 'Esatta',
        hybridDesc: 'Mantiene la tastiera boot e il mouse relativo, per i dispositivi compatibili.',
        exactDesc: 'Sostituisce ogni funzione USB del NanoKVM con il dispositivo importato.',
        hybridWarning: 'La modalità ibrida mantiene disponibili tastiera e mouse relativo',
        hybridWarningDesc:
          'Archiviazione, rete USB e puntatore assoluto si disconnettono mentre la funzione importata è attiva.',
        hidWarning: 'Avviare il passthrough cede tastiera, mouse e supporti virtuali',
        hidWarningDesc:
          'NanoKVM ha un solo controller di dispositivo USB e il proxy lo occupa per intero, quindi mentre una sessione è attiva l’host remoto vede il dispositivo inoltrato al posto di tastiera, mouse e supporti virtuali di NanoKVM. Tornano da soli non appena la sessione viene fermata. Questa interfaccia web non è coinvolta, quindi puoi sempre fermare la sessione da questa pagina.',
        hidWarningSafeDesc:
          'Il NanoKVM ha un solo controller di dispositivo USB e il proxy lo occupa tutto, quindi durante una sessione l’host remoto vede il dispositivo inoltrato al posto di tastiera, mouse e supporti virtuali del NanoKVM. Tornano quando la sessione si ferma.',
        isoLabel: 'Consenti trasferimenti isocroni',
        isoHint:
          'Lascia passare webcam, microfoni e altri dispositivi in streaming. Nessuno ha misurato cosa regge questo hardware.',
        isoWarning:
          'Lo streaming isocrono non è collaudato qui e può trattenere tastiera e mouse finché non fermi la sessione',
        info: {
          title: 'Informazioni',
          hybrid:
            'La modalità ibrida mantiene disponibili la tastiera e il mouse relativo. Archiviazione, rete USB e puntatore assoluto si disconnettono mentre il dispositivo importato è attivo.',
          exact:
            'La modalità esatta sostituisce ogni funzione USB del NanoKVM con il dispositivo importato. Tastiera, mouse e supporti virtuali tornano da soli quando la sessione si ferma.',
          udc: 'Il NanoKVM ha un solo controller di dispositivo USB e il proxy lo occupa tutto: per questo le funzioni qui sopra scompaiono per tutta la durata di una sessione.',
          web: 'Questa interfaccia web non è coinvolta, quindi puoi sempre fermare una sessione da questa pagina.',
          network:
            'Avvia il passthrough tramite Ethernet o Wi-Fi. Avviarlo dalla rete USB del NanoKVM viene rifiutato, perché quella connessione sparirebbe.',
          iso: 'Webcam, microfoni e altri dispositivi isocroni vengono rifiutati finché non consenti i trasferimenti isocroni. Quella strada funziona ma non è mai stata misurata su questo hardware: considera il suo throughput sconosciuto.',
          camera:
            'La fotocamera e il microfono del browser, sotto Dispositivi, restano il modo collaudato per darne uno alla macchina remota.'
        },
        session: 'Sessione',
        activeDesc: 'Un dispositivo è importato e il proxy tiene il controller USB.',
        inactiveDesc:
          'Nessuna sessione attiva. Tastiera, mouse e supporti virtuali funzionano normalmente.',
        device: 'Dispositivo',
        busId: 'ID bus',
        speed: 'Velocità',
        exporter: 'Exporter',
        local: 'Importato come',
        localValue: 'Bus {{bus}}, indirizzo {{address}}',
        udc: 'Controller USB',
        pid: 'PID del proxy',
        startedAt: 'Avviata',
        isoDevice:
          'Questo dispositivo trasmette su endpoint isocroni, cosa mai misurata su questo hardware',
        exporterLabel: 'Indirizzo dell’exporter',
        exporterHint: 'Host e porta che NanoKVM contatta. Con il tunnel qui sotto è {{exporter}}.',
        busIdLabel: 'ID bus sulla tua macchina',
        busIdHint: 'Il busid che usbip list -l stampa per il dispositivo, ad esempio {{example}}.',
        start: 'Avvia passthrough',
        stop: 'Ferma passthrough',
        startTitle: 'Avviare il passthrough USB?',
        startDevice: 'NanoKVM importerà {{busId}} da {{exporter}}.',
        startHid:
          'Tastiera USB, mouse e supporti virtuali smettono di funzionare per tutta la durata della sessione e ripartono da soli quando la fermi.',
        startIso:
          'Le webcam e gli altri dispositivi isocroni richiedono di attivare l’interruttore isocrono prima di avviare.',
        startWeb:
          'Questa interfaccia web continua a funzionare, quindi puoi fermare la sessione da questa pagina in qualsiasi momento.',
        startNetwork:
          'Usa questa pagina tramite Ethernet o Wi-Fi. Avviarla dalla rete USB del NanoKVM viene rifiutato perché quella connessione sparirebbe.',
        okBtn: 'Avvia',
        cancelBtn: 'Annulla',
        instructions: 'Sulla tua macchina',
        instructionsDesc:
          'Per scelta progettuale non c’è alcun agente da installare. Esegui questi normali comandi usbip sulla macchina a cui è collegato il dispositivo.',
        copyFailed: 'Copia non riuscita. Copia il comando manualmente.',
        copyInsecure:
          'Questa pagina non è servita via HTTPS, quindi il browser blocca la copia. Copia il comando manualmente, oppure abilita HTTPS in Impostazioni, Rete.',
        directNote:
          'Senza tunnel, usbipd deve essere raggiungibile sulla tua rete e l’indirizzo dell’exporter qui sopra deve indicarlo. usbip trasporta il dispositivo in chiaro, quindi è preferibile il tunnel.',
        steps: {
          modprobe: {
            title: 'Carica il driver lato exporter',
            desc: 'usbip-host è ciò che permette al kernel di cedere un dispositivo locale. Non viene caricato di default.'
          },
          list: {
            title: 'Trova il dispositivo',
            desc: 'Elenca ogni dispositivo locale con il suo busid e la coppia produttore:prodotto. Annota il busid di quello che ti serve.'
          },
          bind: {
            title: 'Collegalo a usbip',
            desc: 'Toglie il dispositivo al suo driver abituale, quindi smette di funzionare su questa macchina finché non lo scolleghi.'
          },
          serve: {
            title: 'Pubblicalo',
            desc: 'usbipd resta in primo piano e attende che NanoKVM importi il dispositivo.',
            notice:
              'Il usbipd standard non ha un’opzione per l’indirizzo di ascolto e ascolta su tutte le interfacce. Tieni la porta {{port}} chiusa sul firewall e lascia che la raggiunga solo il tunnel qui sotto.'
          },
          tunnel: {
            title: 'Puntalo a NanoKVM',
            desc: 'Un tunnel SSH inverso: la porta {{port}} sul loopback di NanoKVM diventa il usbipd di questa macchina. Lascialo attivo per tutta la sessione.'
          },
          exporter: {
            title: 'Usa questo come exporter',
            desc: 'Inseriscilo nel campo dell’exporter qui sopra, indica l’ID bus e avvia la sessione.'
          },
          unbind: {
            title: 'Restituisci il dispositivo',
            desc: 'Dopo aver fermato la sessione, questo restituisce il dispositivo al suo driver abituale su questa macchina.'
          }
        }
      },
      mcp: {
        title: 'Servizio MCP',
        service: 'Controllo remoto MCP',
        serviceDesc:
          'Consenti ai client MCP attendibili di controllare tastiera e mouse e acquisire schermate',
        securityWarning:
          'Chiunque disponga di questa chiave API può controllare l’host remoto e visualizzarne lo schermo. Usa HTTPS e abilita il servizio solo su reti attendibili.',
        endpoint: 'Endpoint',
        apiKey: 'Chiave API',
        regenerateConfirmTitle: 'Rigenerare la chiave API MCP?',
        regenerateConfirmDesc: 'La chiave attuale smetterà immediatamente di funzionare.',
        enableConfirmTitle: 'Abilitare il controllo MCP esterno?',
        enableConfirmDesc:
          'L’abilitazione di MCP arresterà PicoClaw e chiuderà tutte le sessioni PicoClaw attive.',
        failed: 'Operazione MCP non riuscita',
        copyFailed: 'Copia non riuscita. Copia manualmente.',
        copyInsecure:
          'Questa pagina non è servita via HTTPS, quindi il browser blocca la copia. Copia manualmente, oppure abilita HTTPS in Impostazioni, Rete.',
        okBtn: 'Conferma',
        cancelBtn: 'Annulla'
      },
      about: {
        title: 'Informazioni su NanoKVM',
        information: 'Informazioni',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Versione Applicazione',
        applicationTip: 'Versione dell’applicazione web NanoKVM',
        image: 'Versione Immagine',
        imageTip: 'Versione dell’immagine di sistema NanoKVM',
        deviceKey: 'Chiave Dispositivo',
        community: 'Comunità',
        hostname: 'Nome host',
        hostnameUpdated: 'Nome host aggiornato. Riavviare per applicare.',
        ipType: {
          Wired: 'Cablato',
          Wireless: 'Senza fili',
          Other: 'Altro'
        }
      },
      appearance: {
        title: 'Aspetto',
        display: 'Schermo',
        language: 'Lingua',
        languageDesc: "Seleziona la lingua per l'interfaccia",
        webTitle: 'Titolo web',
        webTitleDesc: 'Personalizza il titolo della pagina web',
        favicon: 'Favicon',
        faviconDesc: "Personalizza l'icona della scheda del browser",
        faviconPlaceholder: "URL dell'immagine",
        faviconUpload: 'Carica',
        faviconReset: 'Ripristina',
        faviconCustom: 'Icona personalizzata',
        faviconBoot: 'Icona da /boot/logo.ico',
        faviconDefault: 'Icona predefinita',
        faviconOverridesBoot: 'Sostituisce /boot/logo.ico',
        faviconErrUrl: 'Inserisci un indirizzo immagine http:// o https://',
        faviconErrFetch: "Il dispositivo non è riuscito a scaricare l'immagine",
        faviconErrLarge: 'Immagine troppo grande. Il limite è 256 KB',
        faviconErrType: 'Immagine non supportata. Usa .ico, .png, .svg, .gif o .jpg',
        faviconErrSave: "Salvataggio dell'icona non riuscito",
        menuBar: {
          title: 'Barra dei menu',
          mode: 'Modalità di visualizzazione',
          modeDesc: 'Visualizza la barra dei menu sullo schermo',
          modeOff: 'Spento',
          modeAuto: 'Nascondi automaticamente',
          modeAlways: 'Sempre visibile',
          keyboardLedStatus: 'Indicatori di blocco della tastiera',
          keyboardLedStatusDesc:
            'Mostra lo stato di Bloc Num, Bloc Maiusc e Bloc Scorr del computer remoto',
          icons: 'Icone dei sottomenu',
          iconsDesc: 'Visualizza le icone dei sottomenu nella barra dei menu'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Stato dei blocchi della tastiera remota',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Bloc Num',
        numLockShort: 'Num',
        capsLock: 'Bloc Maiusc',
        capsLockShort: 'Mai',
        scrollLock: 'Bloc Scorr',
        scrollLockShort: 'Scorr',
        on: 'Attivo',
        off: 'Disattivo',
        unknown: 'Sconosciuto'
      },
      device: {
        title: 'Dispositivo',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Mai',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 ora'
        },
        ssh: {
          description: 'Abilita SSH accesso remoto',
          tip: "Imposta una password complessa prima dell'abilitazione (Account - Modifica password)"
        },
        advanced: 'Impostazioni avanzate',
        swap: {
          title: 'Scambia',
          disable: 'Disabilita',
          description: 'Imposta la dimensione del file di scambio',
          tip: 'Abilitare questa funzione potrebbe ridurre la durata utile della tua scheda SD!'
        },
        mouseJiggler: {
          title: 'Muovi il mouse',
          description: "Impedisce la sospensione dell'host remoto",
          disable: 'Disabilita',
          absolute: 'Modalità assoluta',
          relative: 'Modalità relativa'
        },
        mdns: {
          description: 'Abilita il servizio di rilevamento mDNS',
          tip: 'Spegnerlo se non è necessario'
        },
        hdmi: {
          description: 'Abilita HDMI/monitora uscita',
          idleTimeoutTitle: 'Timeout cattura inattiva',
          idleTimeoutDescription:
            'Interrompi la cattura HDMI dopo che non ci sono visualizzatori attivi per',
          minutes: 'min'
        },
        autostart: {
          title: 'Impostazioni script di avvio automatico',
          description:
            "Gestisce gli script che vengono eseguiti automaticamente all'avvio del sistema",
          new: 'Nuovo',
          deleteConfirm: 'Sei sicuro di voler eliminare questo file?',
          yes: 'Sì',
          no: 'No',
          scriptName: 'Nome script di avvio automatico',
          scriptContent: 'Contenuto script di avvio automatico',
          settings: 'Impostazioni'
        },
        hidOnly: 'HID-Solo modalità',
        hidOnlyDesc:
          'Smette di emulare i dispositivi virtuali, mantenendo solo il controllo di base HID',
        disk: 'Disco virtuale',
        diskDesc: 'Mount virtual U-disk on the remote host',
        rebindNotice:
          'Cambiare questo interruttore rienumera il dispositivo USB, quindi il target perde per un attimo i dispositivi virtuali e la rete USB.',
        media: {
          title: 'Slot per fotocamera e microfono',
          desc: 'Dichiara i dispositivi multimediali che i browser possono occupare. Il budget degli endpoint viene verificato quando si applica il profilo USB. Il salvataggio rienumera il dispositivo e disconnette i browser collegati.',
          cameras: 'Fotocamere',
          microphones: 'Microfoni',
          name: 'Nome',
          namePlaceholder: 'Mostrato sull’host di destinazione',
          addCamera: 'Aggiungi fotocamera',
          addMicrophone: 'Aggiungi microfono',
          remove: 'Rimuovi',
          cameraDefault: 'Fotocamera NanoKVM {{index}}',
          microphoneDefault: 'Microfono NanoKVM {{index}}',
          nameRequired: 'Ogni slot richiede un nome.',
          budgetHint:
            "I sei endpoint USB IN sono un limite fisso dell'hardware. Metti tastiera, mouse e puntatore assoluto su una sola interfaccia HID in Presentazione USB, oppure disattiva qui il disco virtuale o, in Rete, l’adattatore di rete USB.",
          unsupported:
            'Questo kernel non può assegnare un nome ai dispositivi multimediali, quindi gli host mostrano il nome predefinito.',
          save: 'Salva slot',
          disconnect: 'Disconnetti',
          disconnectAll: 'Disconnetti tutte le sorgenti',
          limit: 'Gli slot per fotocamera e microfono devono essere in totale otto o meno.',
          failed: 'Impossibile aggiornare gli slot multimediali.'
        },
        reboot: 'Riavvia',
        rebootDesc: 'Sei sicuro di voler riavviare NanoKVM?',
        okBtn: 'Sì',
        cancelBtn: 'No'
      },
      network: {
        title: 'Rete',
        wifi: {
          title: 'Wi-Fi',
          description: 'Configura Wi-Fi',
          apMode: 'La modalità AP è attiva, connettiti al Wi-Fi scansionando il codice QR',
          connect: 'Connetti Wi-Fi',
          connectDesc1: 'Inserisci SSID e password della rete',
          connectDesc2: 'Inserisci la password per unirti a questa rete',
          disconnect: 'Vuoi davvero disconnettere la rete?',
          failed: 'Connessione non riuscita, riprova.',
          ssid: 'Nome',
          password: 'Password',
          joinBtn: 'Connetti',
          confirmBtn: 'OK',
          cancelBtn: 'Annulla'
        },
        tls: {
          description: 'Abilita protocollo HTTPS',
          tip: "Attenzione: l'uso di HTTPS può aumentare la latenza, soprattutto in modalità video MJPEG."
        },
        usb: {
          title: 'Adattatore di rete USB',
          desc: 'Dà al computer controllato una scheda di rete via USB',
          type: 'Tipo di adattatore',
          typeDesc: 'NCM per sistemi moderni, RNDIS per Windows datati'
        },
        bridge: {
          title: 'L’adattatore è collegato a',
          lan: 'La tua rete',
          kvmOnly: 'Solo NanoKVM',
          lanDesc:
            'Il computer entra nella tua rete dalla porta Ethernet del NanoKVM, con un proprio indirizzo dal router.',
          kvmOnlyDesc:
            'Il computer riceve l’indirizzo dal NanoKVM e raggiunge il NanoKVM, ma nulla oltre.',
          loading: 'Caricamento...',
          state: 'Stato',
          states: {
            disabled: 'Solo NanoKVM',
            enabled: 'La tua rete',
            rolledBack: 'Ripristinato',
            failed: 'Non riuscito',
            pending: 'In corso'
          },
          uplink: 'Uplink',
          ports: 'Porte',
          up: 'attiva',
          down: 'inattiva',
          noLink: 'nessun link',
          enableTitle: 'Collegare il computer alla tua rete?',
          disableTitle: 'Limitare il computer al solo NanoKVM?',
          reconnect:
            'La connessione di gestione si interromperà brevemente e si riconnetterà mentre l’indirizzo viene spostato.',
          rollback:
            'Se la verifica non riesce, la configurazione precedente viene ripristinata automaticamente.',
          enableBtn: 'Collega alla mia rete',
          disableBtn: 'Solo NanoKVM',
          cancelBtn: 'Annulla',
          interrupted:
            'La connessione si è interrotta durante l’applicazione. Verifica dello stato attuale in corso.',
          pendingNotice:
            'Una modifica del bridge è ancora in corso o è stata interrotta prima di completarsi.',
          revert: 'Ripristina la configurazione precedente',
          rolledBackNotice:
            'L’ultima modifica è stata annullata ed è stata ripristinata la configurazione precedente.',
          verifyFailed: 'Verifica non riuscita: {{gates}}',
          gates: {
            address: 'indirizzo',
            gateway: 'gateway',
            inbound: 'connessione in entrata'
          },
          inboundWeak:
            'La verifica in entrata è passata solo perché NanoKVM si è connesso a se stesso. Questo dimostra che il servizio web è in ascolto ed è raggiungibile localmente, non che una richiesta dalla rete arrivi davvero.',
          noCarrier:
            'Nessun link su {{port}}. Il bridge non ha alcun percorso verso la rete finché non si collega un cavo.',
          loop: 'Il router viene appreso anche su {{port}}, quindi quella porta è un secondo percorso verso la stessa rete. Lo spanning tree è disattivato, perciò qui nulla interromperà il loop: scollega uno dei due percorsi.',
          failedNotice:
            'Non è stato possibile annullare l’ultima modifica. NanoKVM potrebbe essere raggiungibile solo tramite l’AP Wi-Fi o una console seriale.'
        },
        dns: {
          title: 'DNS',
          description: 'Configura i server DNS per NanoKVM',
          mode: 'Modalità',
          dhcp: 'DHCP',
          manual: 'Manuale',
          add: 'Aggiungi DNS',
          save: 'Salva',
          invalid: 'Inserisci un indirizzo IP valido',
          noDhcp: 'Nessun DNS DHCP è attualmente disponibile',
          saved: 'Impostazioni DNS salvate',
          saveFailed: 'Impossibile salvare le impostazioni DNS',
          unsaved: 'Modifiche non salvate',
          maxServers: 'Sono consentiti al massimo {{count}} server DNS',
          dnsServers: 'Server DNS',
          dhcpServersDescription: 'I server DNS vengono ottenuti automaticamente da DHCP',
          manualServersDescription: 'I server DNS possono essere modificati manualmente',
          networkDetails: 'Dettagli rete',
          interface: 'Interfaccia',
          ipAddress: 'Indirizzo IP',
          subnetMask: 'Subnet mask',
          router: 'Router',
          none: 'Nessuno'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'Server VNC',
        description:
          'Permette a qualsiasi client VNC di vedere lo schermo remoto e usare tastiera e mouse, accedendo con il tuo account NanoKVM',
        port: 'Porta',
        portDescription: 'Collegati a questa porta sull’indirizzo del NanoKVM'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Ottimizzazione memoria',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Scambia memoria',
          tip: 'Se i problemi persistono dopo aver abilitato l\'ottimizzazione della memoria, provare ad abilitare la memoria di scambio. Ciò imposta la dimensione del file di scambio su 256MB per impostazione predefinita, che può essere regolata in "Impostazioni > Dispositivo".'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Caricamento...',
        notInstall: 'Tailscale non trovato! Per favore, installa.',
        install: 'Installa',
        installing: 'Installazione in corso',
        failed: 'Installazione fallita',
        retry: 'Riprova aggiornando la pagina o installa manualmente',
        download: 'Scarica il',
        package: 'pacchetto di installazione',
        unzip: 'e decomprimilo',
        upTailscale: 'Carica tailscale nella directory /usr/bin/ del NanoKVM',
        upTailscaled: 'Carica tailscaled nella directory /usr/sbin/ del NanoKVM',
        refresh: 'Aggiorna la pagina corrente',
        notRunning: 'Tailscale non è in esecuzione. Per favore avvialo per continuare.',
        run: 'Inizio',
        notLogin:
          'Il dispositivo non è ancora stato associato. Effettua il login e associa questo dispositivo al tuo account.',
        urlPeriod: 'Questo URL è valido per 10 minuti',
        login: 'Accedi',
        loginSuccess: 'Accesso riuscito',
        enable: 'Abilita Tailscale',
        deviceName: 'Nome Dispositivo',
        deviceIP: 'IP Dispositivo',
        account: 'Account',
        logout: 'Disconnetti',
        logoutDesc: 'Sei sicuro di voler uscire?',
        uninstall: 'Disinstalla Tailscale',
        uninstallDesc: 'Sei sicuro di voler disinstallare Tailscale?',
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Caricamento...',
        notInstall: 'Non installato',
        notConfigured: 'Non configurato',
        stopped: 'Arrestato',
        running: 'In esecuzione',
        connected: 'Connesso',
        error: 'Errore',
        atBoot: 'si avvia all’accensione',
        notAtBoot: 'non si avvia all’accensione',
        arguments: 'Argomenti',
        argumentsTip: 'Argomenti da riga di comando passati al servizio all’avvio.',
        env: 'Variabili d’ambiente',
        envKey: 'Nome',
        envValue: 'Valore',
        envAdd: 'Aggiungi variabile',
        envRemove: 'Rimuovi',
        configured: 'Configurata',
        save: 'Salva',
        saved: 'Configurazione salvata',
        start: 'Avvia',
        stop: 'Arresta',
        restart: 'Riavvia',
        logs: 'Log',
        logsEmpty: 'Nessun log al momento',
        refresh: 'Aggiorna',
        binary: 'Binario',
        binaryShipped: 'Incluso nel firmware',
        binaryCustom: 'Binario personalizzato',
        binaryUpload: 'Carica binario',
        binaryRevert: 'Ripristina il binario incluso',
        binaryRevertDesc:
          'Eliminare il binario caricato e ripristinare quello incluso nel firmware?',
        serverWarning: 'Un server senza restrizioni funziona da proxy aperto',
        noHealthSignal:
          'Questo servizio non riporta alcun segnale di stato, quindi NanoKVM sa solo che il processo è in esecuzione, non se il tunnel è connesso.',
        memoryWarning: 'Eseguire più servizi di accesso remoto insieme può esaurire la memoria',
        resources: 'Risorse',
        memory: {
          title: 'Limite di memoria',
          description:
            'Limita l’heap Go di newt a {{limit}} MiB dal prossimo riavvio. È il suo limite, non quello di Tailscale; da spento resta il valore predefinito di Go, con GOGC=50 in entrambi i casi.',
          noRuntime:
            'wstunnel è scritto in Rust: nessun garbage collector e nessun limite di heap da impostare, e i suoi thread di lavoro seguono già l’unica CPU del dispositivo.',
          notApplicable: 'Non applicabile'
        },
        swap: {
          title: 'File di swap',
          description:
            'Aggiunge un file di swap da 256 MB sulla scheda SD. Vale per tutto il sistema: lo stesso swap serve Tailscale, il server KVM e tutto il resto sul dispositivo.'
        },
        okBtn: 'Sì',
        cancelBtn: 'No'
      },
      update: {
        title: 'Controlla Aggiornamenti',
        queryFailed: 'Impossibile ottenere la versione',
        updateFailed: 'Aggiornamento fallito. Riprova.',
        isLatest: 'Hai già la versione più recente.',
        rebooting:
          'Installazione del nuovo kernel e riavvio in corso. Può richiedere alcuni minuti; non togliere l’alimentazione.',
        kernelUpdate:
          'Questo aggiornamento installa il kernel {{version}}. Il dispositivo si riavvia e torna da solo al kernel attuale se il nuovo non si avvia.',
        rolledBack:
          'Il kernel {{version}} non si è avviato e il dispositivo è tornato al kernel precedente.',
        available: 'Un aggiornamento è disponibile. Sei sicuro di voler aggiornare?',
        updating: 'Aggiornamento avviato. Attendere prego...',
        confirm: 'Conferma',
        cancel: 'Annulla',
        preview: 'Anteprima aggiornamenti',
        previewDesc: "Ottieni l'accesso anticipato a nuove funzionalità e miglioramenti",
        previewTip:
          'Tieni presente che le versioni di anteprima possono contenere bug o funzionalità incomplete!',
        customServer: {
          title: 'Server di aggiornamento personalizzato',
          desc: 'Cerca e scarica gli aggiornamenti online da un server specificato',
          invalidUrl:
            'Inserisci una directory server HTTP o HTTPS valida, senza parametri di query, frammenti o latest.json.',
          loadFailed: 'Impossibile caricare la configurazione del server di aggiornamento.',
          saveFailed: 'Impossibile salvare la configurazione del server di aggiornamento.',
          saved: 'Configurazione del server di aggiornamento salvata.',
          save: 'Salva',
          confirmTitle: 'Utilizzare un server di aggiornamento personalizzato?',
          confirmDesc:
            'SHA-512 verifica soltanto che il pacchetto corrisponda al manifesto fornito da questo server. Non garantisce che il pacchetto sia una versione ufficiale di NanoKVM. Un server difettoso o dannoso può rendere inutilizzabile il dispositivo, causare la perdita di dati o compromettere il sistema.',
          confirm: 'Utilizza comunque',
          previewDisabled:
            'Gli aggiornamenti in anteprima non sono disponibili quando è attivo un server di aggiornamento personalizzato.'
        },
        offline: {
          kernelNotice:
            'Questo pacchetto contiene un kernel. Viene scritto nello slot di riserva e il dispositivo si riavvia per provarlo; se non torna, ritorna da solo al kernel attuale.',
          kernelConfirm: 'Installa kernel',
          kernelCancel: 'Annulla',
          title: 'Aggiornamenti offline',
          desc: 'Aggiornamento tramite pacchetto di installazione locale',
          upload: 'Carica',
          checksumPlaceholder: 'Checksum SHA-256 (facoltativo)',
          invalidChecksum: 'Il checksum SHA-256 deve contenere 64 caratteri esadecimali.',
          checksumMismatch:
            'La verifica SHA-256 non è riuscita. Il pacchetto potrebbe essere danneggiato.',
          invalidName: 'Formato nome file non valido. Si prega di scaricare dalle versioni GitHub.',
          updateFailed: 'Aggiornamento fallito. Riprova.'
        }
      },
      account: {
        title: 'Account',
        webAccount: 'Nome account web',
        role: 'Ruolo',
        roles: {
          admin: 'Amministratore',
          user: 'Utente'
        },
        password: 'Password',
        updateBtn: 'Update',
        logoutBtn: 'Esci',
        logoutDesc: 'Sei sicuro di voler uscire?',
        okBtn: 'Sì',
        cancelBtn: 'No',
        users: {
          title: 'Utenti',
          create: 'Crea utente',
          enabled: 'Abilitato',
          disabled: 'Disabilitato',
          deviceOwner: 'Proprietario del dispositivo',
          resetPassword: 'Reimposta password',
          delete: 'Elimina',
          deleteConfirm: 'Eliminare questo utente e revocare tutte le sue sessioni?',
          created: 'Utente creato',
          deleted: 'Utente eliminato',
          passwordUpdated: 'Password aggiornata',
          loadFailed: 'Impossibile caricare gli utenti',
          saveFailed: 'Impossibile salvare l’utente',
          deleteFailed: 'Impossibile eliminare l’utente'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistente',
      empty: "Apri il pannello e avvia un'attività da iniziare.",
      inputPlaceholder: 'Descrivi cosa vuoi che PicoClaw faccia',
      newConversation: 'Nuova conversazione',
      processing: 'In elaborazione...',
      agent: {
        defaultTitle: 'Assistente generale',
        defaultDescription: "Chat generale, ricerca e aiuto nell'area di lavoro.",
        kvmTitle: 'Controllo remoto',
        kvmDescription: "Gestisci l'host remoto tramite NanoKVM.",
        switched: "Ruolo dell'agente cambiato",
        switchFailed: "Impossibile cambiare il ruolo dell'agente"
      },
      send: 'Invia',
      cancel: 'Annulla',
      status: {
        connecting: 'Connessione al gateway...',
        connected: 'Sessione PicoClaw connessa',
        disconnected: 'Sessione PicoClaw chiusa',
        stopped: 'Richiesta di interruzione inviata',
        runtimeStarted: 'Runtime PicoClaw avviato',
        runtimeStartFailed: 'Impossibile avviare il runtime PicoClaw',
        runtimeStopped: 'Runtime PicoClaw interrotto',
        runtimeStopFailed: 'Impossibile arrestare il runtime di PicoClaw',
        controlSwitchedToMCP: 'Controllo trasferito al servizio MCP esterno'
      },
      connection: {
        runtime: {
          checking: 'Controllo',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime pronto',
          stopped: 'Runtime interrotto',
          blockedByMCP: 'Il controllo MCP esterno è attivo',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime non disponibile',
          configError: 'Errore di configurazione'
        },
        transport: {
          connecting: 'Connessione',
          connected: 'Connesso',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Inattivo',
          busy: 'Occupato'
        }
      },
      message: {
        toolAction: 'Azione',
        observation: 'Osservazione',
        screenshot: 'Schermata'
      },
      overlay: {
        locked: "PicoClaw sta controllando il dispositivo. L'immissione manuale è in pausa."
      },
      control: {
        picoclaw: 'Controllo dispositivo: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Controllo dispositivo: MCP esterno',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Controllo dispositivo: disattivato',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Concedi controllo',
        release: 'Rilascia',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Controllo PicoClaw concesso',
        released: 'Controllo PicoClaw rilasciato',
        grantFailed: 'Impossibile concedere il controllo PicoClaw',
        releaseFailed: 'Impossibile rilasciare il controllo PicoClaw',
        grantConfirmTitle: 'Passare il controllo del dispositivo a PicoClaw?',
        grantConfirmDesc: 'Le scritture del dispositivo MCP esterno saranno interrotte.'
      },
      install: {
        install: 'Installa PicoClaw',
        installing: 'Installazione PicoClaw',
        success: 'PicoClaw installato correttamente',
        failed: 'Impossibile installare PicoClaw',
        uninstalling: 'Disinstallazione del runtime in corso...',
        uninstalled: 'Runtime disinstallato correttamente.',
        uninstallFailed: 'Disinstallazione non riuscita.',
        requiredTitle: 'PicoClaw non è installato',
        requiredDescription: 'Installa PicoClaw prima di avviare il runtime PicoClaw.',
        progressDescription: 'PicoClaw è in fase di download e installazione.',
        stages: {
          preparing: 'Preparazione',
          downloading: 'Download in corso',
          extracting: 'Estrazione',
          verifying: 'Verifica in corso',
          installing: 'Installazione in corso',
          installed: 'Installato',
          install_timeout: 'Timeout',
          install_failed: 'Non riuscito'
        }
      },
      model: {
        requiredTitle: 'È richiesta la configurazione del modello',
        requiredDescription:
          'Configura il modello PicoClaw prima di utilizzare la chat di PicoClaw.',
        docsTitle: 'Guida alla configurazione',
        docsDesc: 'Modelli e protocolli supportati',
        menuLabel: 'Configura modello',
        modelIdentifier: 'Identificatore del modello',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Chiave API',
        apiKeyPlaceholder: 'Inserisci la chiave API del modello',
        save: 'Salva',
        saving: 'Salvataggio',
        saved: 'Configurazione del modello salvata',
        saveFailed: 'Impossibile salvare la configurazione del modello',
        invalid: 'Identificatore del modello, API Base URL e chiave API sono obbligatori'
      },
      uninstall: {
        menuLabel: 'Disinstalla',
        confirmTitle: 'Disinstalla PicoClaw',
        confirmContent:
          "Sei sicuro di voler disinstallare PicoClaw? Ciò eliminerà l'eseguibile e tutti i file di configurazione.",
        confirmOk: 'Disinstalla',
        confirmCancel: 'Annulla'
      },
      history: {
        title: 'Cronologia',
        loading: 'Caricamento sessioni...',
        emptyTitle: 'Nessuna cronologia ancora',
        emptyDescription: 'Le sessioni PicoClaw precedenti verranno visualizzate qui.',
        loadFailed: 'Impossibile caricare la cronologia della sessione',
        deleteFailed: 'Impossibile eliminare la sessione',
        deleteConfirmTitle: 'Elimina sessione',
        deleteConfirmContent: 'Sei sicuro di voler eliminare "{{title}}"?',
        deleteConfirmOk: 'Elimina',
        deleteConfirmCancel: 'Annulla',
        messageCount_one: '{{count}} messaggio',
        messageCount_other: '{{count}} messaggi',
        messageCount: '{{count}} messaggi'
      },
      config: {
        startRuntime: 'Avvia PicoClaw',
        stopRuntime: 'Arresta PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Trasferire il controllo a PicoClaw?',
        enableConfirmDesc: 'L’avvio di PicoClaw disabiliterà il servizio MCP esterno.',
        enableConfirmOk: 'Avvia PicoClaw',
        enableConfirmCancel: 'Annulla',
        title: 'Avvia PicoClaw',
        description: "Avvia il runtime per iniziare a utilizzare l'assistente PicoClaw.",
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Si è verificato un problema',
      refresh: 'Aggiorna'
    },
    fullscreen: {
      toggle: 'Attiva/disattiva schermo intero'
    },
    menu: {
      collapse: 'Comprimi menu',
      expand: 'Espandi il menu'
    }
  }
};

export default it;
