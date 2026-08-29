const nb = {
  translation: {
    head: {
      desktop: 'Eksternt skrivebord',
      login: 'Logg inn',
      changePassword: 'Endre passord',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Logg inn',
      placeholderUsername: 'Brukernavn',
      placeholderPassword: 'Passord',
      placeholderCurrentPassword: 'Nåværende passord',
      placeholderPassword2: 'Oppgi passord igjen',
      noEmptyUsername: 'Brukernavn påkrevd',
      noEmptyPassword: 'Passord påkrevd',
      passwordLength: 'Passordet må være mellom 8 og 72 tegn',
      noAccount:
        'Kunne ikke hente brukerinformasjon. Vennligst last inn siden på nytt eller gjenopprett passord',
      invalidUser: 'Ugyldig brukernavn eller passord',
      locked: 'For mange pålogginger, vennligst prøv igjen senere',
      globalLocked: 'System under beskyttelse, prøv igjen senere',
      error: 'Uventet feil',
      invalidCurrentPassword: 'Det nåværende passordet er feil',
      changePassword: 'Endre passord',
      changePasswordDesc:
        'For sikkerheten til enheten, vennligst endre passordet ditt for web-innlogging.',
      differentPassword: 'Passordene er ikke like',
      illegalUsername: 'Brukernavn inneholder tegn som ikke er tillat',
      illegalPassword: 'Passord inneholder tegn som ikke er tillat',
      forgetPassword: 'Glemt passord',
      ok: 'Ok',
      cancel: 'Avbryt',
      loginButtonText: 'Logg inn',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'Se dette dokumentet for detaljerte trinn:',
        reset3: 'Standard webkonto:',
        reset4: 'Standard SSH-konto:',
        change1: 'Merk at denne handlingen endrer følgende passord:',
        change2: 'Passord for webinnlogging',
        change3: 'Systemets root-passord (SSH-innloggingspassord)',
        change4: 'For å tilbakestille passordene holder du BOOT-knappen på NanoKVM inne.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Konfigurer Wi-Fi for NanoKVM',
      success: 'Please check the network status of NanoKVM and visit the new IP address.',
      failed: 'Operasjonen mislyktes, prøv igjen.',
      invalidMode:
        'Gjeldende modus støtter ikke nettverksoppsett. Gå til enheten din og aktiver Wi-Fi konfigurasjonsmodus.',
      confirmBtn: 'Ok',
      finishBtn: 'Ferdig',
      ap: {
        authTitle: 'Autentisering kreves',
        authDescription: 'Vennligst skriv inn AP passordet for å fortsette',
        authFailed: 'Ugyldig AP passord',
        passPlaceholder: 'AP passord',
        verifyBtn: 'Bekreft'
      }
    },
    screen: {
      scale: 'Skala',
      title: 'Skjerm',
      video: 'Video-kodek',
      videoDirectTips: 'Aktiver HTTPS i "Innstillinger > Enhet" for å bruke denne modusen',
      resolution: 'Oppløsning',
      controlRegion: {
        title: 'Musekalibrering',
        description:
          'Bruk denne innstillingen når den kontrollerte enheten bruker en oppløsning som ikke er 16:9, og markøren er forskjøvet vannrett eller loddrett.',
        off: 'Av',
        auto: 'Automatisk',
        autoWarning:
          'Kalibreringen kan mislykkes hvis brukerprogrammet har en helt svart bakgrunn.',
        manual: 'Manuell',
        selectedResolution: 'Oppløsning for valgt område',
        unused: 'Ikke i bruk',
        originalResolution: 'Opprinnelig oppløsning',
        selectResolution: 'Velg opprinnelig oppløsning',
        addResolution: 'Legg til egendefinert oppløsning',
        add: 'Legg til',
        duplicateResolution: 'Denne oppløsningen finnes allerede.',
        width: 'Bredde',
        height: 'Høyde',
        apply: 'Beregn og bruk',
        invalidResolution: 'Angi en gyldig opprinnelig oppløsning når videoen er klar.',
        select: 'Velg område',
        clear: 'Gjenopprett automatisk registrering',
        saveFailed: 'Kunne ikke lagre inndataområdet.',
        tooSmall: 'Det valgte området er for lite.',
        previewUnavailable: 'Forhåndsvisning er utilgjengelig',
        clearConfirm: 'Gjenopprette automatisk registrering av svarte kanter?',
        dragHint: 'Dra for å velge området på det eksterne skrivebordet',
        finish: 'Ferdig',
        confirm: 'Bekreft',
        cancel: 'Avbryt'
      },
      auto: 'Automatisk',
      autoTips:
        'Skjermriving eller peker-forskyvning kan oppstå ved enkelte oppløsninger. Prøv å justere den eksterne vertens oppløsning eller skru av automatisk modus.',
      fps: 'FPS',
      customizeFps: 'Tilpass',
      quality: 'Kvalitet',
      qualityLossless: 'Tapsfri',
      qualityHigh: 'Høy',
      qualityMedium: 'Middels',
      qualityLow: 'Lav',
      frameDetect: 'Bildefrekvensoppdagelse',
      frameDetectTip:
        'Kalkuler forskjellen mellom bilder. Stopper overføring av video når det ikke oppdages forskjell på den eksterne vertens skjerm.',
      resetHdmi: 'Tilbakestill HDMI',
      mixedH264: {
        title: 'H.264-strømmekonflikt',
        description:
          'H.264 Direct og H.264 WebRTC brukes samtidig. Dette kan føre til skjermriving eller ødelagt video. Bruk bare én H.264-modus.'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC-tilkobling mislyktes',
        description: 'Kontroller nettverkstilkoblingen eller bytt videomodus.'
      },
      captureStatus: {
        hdmiError: 'HDMI-skjermfeil',
        unsupportedResolution: 'Gjeldende oppløsning støttes ikke',
        retrieving: 'Henter skjermbilde...',
        changingResolution: 'Bytter oppløsning...',
        updateFailed: 'Skjermen kan ikke oppdateres akkurat nå',
        videoError: 'Feil ved videovisning',
        noHdmi: 'Ingen HDMI-signal oppdaget',
        unavailable: 'Skjermen kan ikke vises akkurat nå'
      }
    },
    keyboard: {
      title: 'Åpne tastatur',
      paste: 'Lim inn',
      tips: 'Kun vanlige tegn på tastatur er støttet',
      placeholder: 'Vennligst angi teksten du vil lime inn',
      submit: 'Lim inn',
      virtual: 'Åpne tastatur',
      readClipboard: 'Les fra utklippstavlen',
      clipboardPermissionDenied:
        'Utklippstavle tillatelse nektet. Tillat utklippstavletilgang i nettleseren din.',
      clipboardReadError: 'Kunne ikke lese utklippstavlen',
      dropdownEnglish: 'Engelsk',
      dropdownGerman: 'tysk',
      dropdownFrench: 'Fransk',
      dropdownRussian: 'russisk',
      shortcut: {
        title: 'Snarveier',
        custom: 'Egendefinert',
        capture: 'Klikk her for å ta en snarvei',
        clear: 'Tøm',
        save: 'Lagre',
        captureTips:
          'Registrering av systemtaster (som Windows-tasten) krever fullskjermtillatelse.',
        enterFullScreen: 'Veksle fullskjermmodus.'
      },
      leaderKey: {
        title: 'Leader-tast',
        desc: 'Omgå nettleserrestriksjoner og send systemsnarveier direkte til den eksterne verten.',
        howToUse: 'Hvordan bruke',
        simultaneous: {
          title: 'Samtidig modus',
          desc1: 'Hold Leader-tasten inne, og trykk deretter på snarveien.',
          desc2: 'Intuitivt, men kan komme i konflikt med systemsnarveier.'
        },
        sequential: {
          title: 'Sekvensiell modus',
          desc1:
            'Trykk på Leader-tasten → trykk på snarveien i rekkefølge → trykk på Leader-tasten igjen.',
          desc2: 'Krever flere trinn, men unngår fullstendig systemkonflikter.'
        },
        enable: 'Aktiver Leader-tast',
        tip: 'Når denne tasten er tilordnet som Leader-tast, fungerer den bare som snarveisutløser og mister standardoppførselen.',
        placeholder: 'Trykk på Leader-tasten',
        shiftRight: 'Høyre Shift',
        ctrlRight: 'Høyre Ctrl',
        metaRight: 'Høyre Win',
        submit: 'Lim inn',
        recorder: {
          rec: 'REC',
          activate: 'Aktiver taster',
          input: 'Trykk snarveien...'
        }
      }
    },
    mouse: {
      title: 'Mus',
      cursor: 'Markørstil',
      default: 'Vanlig',
      pointer: 'Hånd',
      cell: 'Celle',
      text: 'Tekst',
      grab: 'Grip',
      hide: 'Skjul',
      mode: 'Modus',
      absolute: 'Absolutt',
      relative: 'Relativ',
      direction: 'Rullehjulretning',
      scrollUp: 'Rull opp',
      scrollDown: 'Rull ned',
      speed: 'Rullehjulhastighet',
      fast: 'Rask',
      slow: 'Sakte',
      requestPointer: 'Bruker relativ modus. Vennligsk klikk på skrivebordet for vise musepeker.',
      resetHid: 'Gjenopprett HID',
      hidOnly: {
        title: 'Kun HID-modus',
        desc: 'Hvis musen og tastaturet slutter å svare og tilbakestilling av HID ikke hjelper, kan det være et kompatibilitetsproblem mellom NanoKVM og enheten. Prøv å aktivere HID-Only-modus for bedre kompatibilitet.',
        tip1: 'Aktivering av HID-Only-modus vil demontere den virtuelle U-disken og det virtuelle nettverket',
        tip2: 'I HID-Only-modus er bildemontering deaktivert',
        tip3: 'NanoKVM vil automatisk starte på nytt etter bytte av modus',
        enable: 'Aktiver HID-Only-modus',
        disable: 'Deaktiver HID-bare-modus'
      }
    },
    image: {
      title: 'Bilder',
      loading: 'Laster...',
      empty: 'Ingen funnet',
      mountMode: 'Monteringsmodus',
      mountFailed: 'Montering feilet',
      mountDesc:
        'På noen systemer er det nødvendig å koble fra den virtuelle disken på den eksterne verten før man kan montere arkivfilen.',
      unmountFailed: 'Avmontering mislyktes',
      unmountDesc:
        'På noen systemer må du manuelt løse ut fra den eksterne verten før du demonterer bildet.',
      refresh: 'Oppdater bildelisten',
      attention: 'Merknad',
      deleteConfirm: 'Er du sikker på at du vil slette dette bildet?',
      okBtn: 'Ja',
      cancelBtn: 'Nei',
      tips: {
        title: 'Hvordan laste opp',
        usb1: 'Koble til NanoKVM-enheten til din datamaskin med USB.',
        usb2: 'Sikre at den virtuelle disken er montert (Innstillinger - Virtuell disk).',
        usb3: 'Åpne den virtuelle disken på datamaskinen din og kopier arkivfilen til rot-mappen på den virtuelle disken.',
        scp1: 'Sikre at NanoKVM-enheten og datamaskinen din er tilkoblet det samme lokale nettverket.',
        scp2: 'Åpne en terminal på datamaskinen din og bruk SCP-kommandoen til å laste opp arkivfilen til mappen /data på NanoKVM-enheten.',
        scp3: 'Eksempel: scp sti-til-din-arkivfil root@din-nanokvm-ip:/data',
        tfCard: 'TF-kort',
        tf1: 'Denne metoden er støttet på datamskiner med Linux',
        tf2: 'Ta TF-kortet ut av NanoKVM-enheten (hvis du har FULL-versjonen, demonter kabinettet først).',
        tf3: 'Sett inn TF-kortet i en kortleser og koble den til datamaskinen din.',
        tf4: 'Kopiér arkivfilen til mappen /data på TF-kortet.',
        tf5: 'Sett inn TF-kortet i NanoKVM-enheten.'
      }
    },
    script: {
      title: 'Skript',
      upload: 'Last opp',
      run: 'Kjør',
      runBackground: 'Kjør i bakgrunnen',
      runFailed: 'Kjøring feilet',
      attention: 'Merknad',
      delDesc: 'Er du sikker på at du vil slette denne filen?',
      confirm: 'Ja',
      cancel: 'Nei',
      delete: 'Slett',
      close: 'Lukk'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM',
      serial: 'Seriell port',
      serialPort: 'Seriell port',
      serialPortPlaceholder: 'Vennligst angi den serielle porten',
      baudrate: 'Baud-rate',
      parity: 'Paritet',
      parityNone: 'Ingen',
      parityEven: 'Lik',
      parityOdd: 'Ulik',
      flowControl: 'Strømningskontroll',
      flowControlNone: 'Ingen',
      flowControlSoft: 'Programvare',
      flowControlHard: 'Maskinvare',
      dataBits: 'Databiter',
      stopBits: 'Stoppbiter',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Sender kommando...',
      sent: 'Kommando sendt',
      input: 'Vennligst angi MAC-adressen',
      ok: 'Ok'
    },
    download: {
      title: 'Bildedaster',
      input: 'Vennligst skriv inn et eksternt bilde URL',
      ok: 'Ok',
      disabled: '/data partisjonen er RO, så vi kan ikke laste ned bildet',
      uploadbox: 'Slipp filen her eller klikk for å velge',
      inputfile: 'Vennligst skriv inn bildefilen',
      NoISO: 'Ingen ISO',
      sha256: 'SHA-256 (valgfritt)',
      sha256Placeholder: 'Skriv inn en SHA-256-kontrollsum på 64 tegn',
      invalidSHA256: 'SHA-256 må være en heksadesimal streng på 64 tegn',
      failed: 'Nedlasting mislyktes',
      success: 'Nedlasting fullført',
      checksumFailed: 'Nedlasting mislyktes: SHA-256-verifisering mislyktes',
      cancel: 'Avbryt',
      cancelFailed: 'Kunne ikke avbryte nedlastingen'
    },
    power: {
      title: 'På-knapp',
      showConfirm: 'Bekreftelse',
      showConfirmTip: 'Strømdrift krever en ekstra bekreftelse',
      reset: 'Reset-knapp',
      power: 'På-knapp',
      powerShort: 'På-knapp (kort trykk)',
      powerLong: 'På-knapp (langt trykk)',
      resetConfirm: 'Fortsette tilbakestilling?',
      powerConfirm: 'Fortsette strømdrift?',
      okBtn: 'Ja',
      cancelBtn: 'Nei'
    },
    devices: {
      title: 'Enheter',
      stale: 'Sanntidsstatusen for enhetene er utilgjengelig. Kobler til igjen.',
      empty:
        'Ingen kamera-, mikrofon- eller høyttalerplasser er satt opp. Legg til en under Innstillinger, Enhet.',
      available: 'Tilgjengelig',
      waiting: 'Verten venter på en kilde',
      hostOpen: 'Vert åpen',
      hostIdle: 'Vert inaktiv',
      hostPlaying: 'Verten spiller av lyd',
      hostSending: 'Vert spiller av',
      sending: 'Sender fra denne nettleseren',
      receiving: 'Spilles av i denne nettleseren',
      black: 'Svart video',
      silence: 'Digital stillhet',
      resuming: 'Venter på å fortsette',
      stop: 'Stopp deling',
      stopListening: 'Stopp lytting',
      disconnect: 'Koble fra',
      takeover: 'Overta',
      refused: 'I bruk av {{owner}} fra {{source}}',
      connectedSources_one: '{{count}} tilkoblet kilde',
      connectedSources_other: '{{count}} tilkoblede kilder',
      connectedSources: '{{count}} tilkoblede kilder',
      connection: {
        connecting: 'Kobler til',
        connected: 'Direkte',
        disconnected: 'Kobler til igjen'
      },
      share: {
        camera: 'Del kamera',
        microphone: 'Del mikrofon',
        speaker: 'Lytt',
        usbDevice: 'Del USB'
      },
      permission: {
        denied: 'Blokkert i nettleserens nettstedsinnstillinger',
        prompt: 'Nettleseren vil spørre om tilgang',
        insecure:
          'Denne siden leveres ikke over HTTPS, så nettleseren blokkerer denne enheten. Slå på HTTPS under Innstillinger, Nettverk.'
      },
      capture: {
        unsupported: 'Denne nettleseren kan ikke ta opp lyd eller video',
        camera: 'Denne nettleseren kan ikke kode kamerabilder',
        microphone: 'Denne nettleseren kan ikke behandle mikrofonlyd',
        speaker: 'Denne nettleseren kan ikke spille av lyd'
      },
      mic: {
        mute: 'Slå av lyden',
        unmute: 'Slå på lyden'
      },
      revoked: {
        released: 'Delingen ble stoppet',
        lease_expired: 'Leien løp ut før denne nettleseren kom tilbake',
        admin_disconnect: 'En administrator koblet fra alle kilder',
        slot_removed: 'Plassen ble fjernet',
        slot_changed: 'Plassen ble konfigurert på nytt',
        taken_over: 'En administrator overtok denne plassen'
      },
      usb: {
        surrendered: 'USB-passthrough holder tastaturet og musen',
        surrenderedDesc:
          'Den eksterne verten ser den importerte enheten i stedet for NanoKVMs tastatur, mus og virtuelle medier. De kommer tilbake når økten stopper.',
        unsupported: 'WebUSB krever en Chromium-nettleser',
        insecure:
          'Denne siden leveres ikke over HTTPS, så nettleseren holder tilbake WebUSB. Slå på HTTPS under Innstillinger, Nettverk.',
        session: 'Videresender {{device}} ({{mode}})',
        idle: 'Ingen passthrough-økt',
        mode: {
          hybrid: 'hybrid',
          exact: 'eksakt'
        }
      }
    },
    settings: {
      title: 'Innstillinger',
      display: {
        title: 'Skjerm',
        loading: 'Laster...',
        active: 'Aktiv EDID',
        activeUnknown:
          'NanoKVM har ikke skrevet noen EDID siden den startet, så det er ukjent hvilken skjerm verten ser.',
        appliedAt: 'Tatt i bruk {{time}}',
        download: 'Last ned',
        downloadBackup: 'Last ned forrige',
        preset: 'Skjermforhåndsvalg',
        presetPlaceholder: 'Velg en skjerm',
        upload: 'Last opp',
        selected: 'Valgt EDID',
        errors: 'Feil',
        warnings: 'Advarsler',
        info: 'Informasjon',
        unknownMonitor: 'Ukjent skjerm',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Lyd',
        audioNo: 'Ingen lyd',
        extensionBlocks: 'Utvidelsesblokker: {{blocks}}',
        apply: 'Bruk',
        applyTitle: 'Vil du bruke denne EDID-en?',
        before: 'Nåværende',
        after: 'Ny',
        hdmiNotice:
          'Videoopptaket stopper mens EDID skrives, og starter av seg selv igjen etterpå.',
        powerCycleNotice:
          'Enheten må kobles fysisk fra strømmen og kobles til igjen før den nye EDID-en trer i kraft.',
        powerCycleUnverified:
          'Skrivingen ble ikke verifisert, så videobrikken beholder det den nå inneholder, helt til denne enheten kobles fysisk fra strømmen og kobles til igjen.',
        applied: 'EDID tatt i bruk og verifisert.',
        applyFailed: 'Det gikk ikke å ta i bruk EDID.',
        busy: 'Videobrikken var opptatt. Prøv igjen.',
        unsupported: 'Denne enheten støtter ikke endring av EDID.',
        toolMissing: 'EDID-verktøyet mangler i denne fastvaren.',
        noAudio: 'Denne EDID-en annonserer ingen lyd, så verten kan slutte å sende lyd.',
        oldVersion: 'Denne EDID-en bruker en eldre versjon enn 1.4.',
        interlaced: 'Foretrukket oppløsning er linjeflettet.',
        tooLarge:
          'Foretrukket oppløsning er større enn 1920x1080 ved 60 Hz, som er mer enn NanoKVM kan fange opp.',
        recovery: 'Gjenoppretting',
        recoveryNeeded:
          'Den siste skrivingen ble ikke verifisert, så EDID-området i videobrikken er i en ukjent tilstand. Gjenopprett fabrikk-EDID for å få en kjent tilstand igjen.',
        recoveryDesc:
          'Skriv en kjent EDID tilbake til videobrikken hvis den du tok i bruk lot verten stå uten bilde.',
        restoreFactory: 'Gjenopprett fabrikk-EDID',
        restoreBackup: 'Gjenopprett forrige EDID',
        restoreTitle: 'Gjenopprette denne EDID-en?',
        restoreFactoryTarget: 'Fabrikk-EDID-en som NanoKVM ble levert med.',
        restoreBackupTarget: 'Den nyeste sikkerhetskopien, tatt i bruk {{time}}.',
        restoreNotice:
          'En gjenoppretting skrives på samme måte som en bruk, med de samme konsekvensene.',
        restored: 'EDID gjenopprettet og verifisert.',
        restoreFailed: 'Gjenoppretting av EDID mislyktes.',
        mismatchTitle: 'Skrevet og lest tilbake',
        mismatchWritten: 'Skrevet',
        mismatchRead: 'Lest tilbake',
        restoreOkBtn: 'Gjenopprett',
        hardware: 'Oppdaget maskinvare: {{hardware}}',
        hardwareUnknown: 'Ukjent',
        confirmWord: 'BRUK',
        confirmPrompt: 'Skriv {{word}} for å aktivere bruk-knappen.',
        okBtn: 'Bruk',
        cancelBtn: 'Avbryt'
      },
      presentation: {
        title: 'USB-presentasjon',
        loading: 'Laster...',
        current: 'Nåværende USB-presentasjon',
        noProfile: 'Ingen profil tatt i bruk',
        linked: 'Tilknyttede funksjoner',
        hostState: 'Vertens USB',
        hostUnbound: 'Kontrolleren er ikke bundet',
        hdmiState: 'HDMI-inngang',
        hdmiSignal: 'Signal til stede',
        hdmiUnreported: 'Ingen melding om opptak ennå',
        endpoints: 'Endepunkter',
        fifos: 'FIFO-plasser',
        pending: 'Ventende endringer',
        pendingEdits: 'Ulagrede identitetsendringer',
        pendingProfile: '{{profile}} er valgt, men ikke tatt i bruk',
        pendingNone: 'Ingen',
        lastApply: 'Siste bruk',
        applyFailed: 'Mislyktes på {{profile}} den {{time}}',
        applyClean: 'Ingen feil registrert',
        lastKnownGood: 'Sist kjente fungerende',
        rollbackTarget: 'Mål for tilbakerulling',
        rollbackNone: 'Ingen',
        powerCyclePending:
          'Kontrolleren ble tatt fra verten. Slå den tilkoblede datamaskinen av og på igjen for å få enheten tilbake.',
        rollback: 'Rull tilbake',
        rollbackTitle: 'Rulle tilbake til {{profile}}?',
        rollbackDesc: 'Gadgeten opplistes på nytt; USB-funksjoner faller ut en kort stund.',
        profile: 'USB-profil',
        builtIn: 'innebygd',
        descriptors: 'deskriptorer',
        imported: 'importert',
        clone: 'Klon',
        cloneTitle: 'Klon denne profilen',
        cloneToEdit:
          'Innebygde profiler forblir skrivebeskyttet. Klon denne profilen for å redigere identiteten.',
        profileName: 'Profilnavn',
        profileNameHint: 'Små bokstaver, tall, punktum, understreker og bindestreker.',
        import: 'Importer pakke',
        export: 'Eksporter pakke',
        delete: 'Slett',
        deleteTitle: 'Slette denne profilen?',
        deleteDesc: 'Slett {{profile}} fra NanoKVM.',
        identity: 'USB-identitet',
        preset: 'Forhåndsvalgt identitet',
        presetPlaceholder: 'Kopier identitet fra en kjent enhet',
        presetHint:
          'Et forhåndsvalg fyller inn Vendor ID, Product ID og de to navnefeltene. Det tar ikke med seg deskriptorer.',
        presetSource: 'Identitet slik den er registrert i {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Denne Vendor ID-en tilhører en annen produsent',
        productId: 'Product ID',
        bcdUSB: 'USB-versjon',
        bcdDevice: 'Enhetsversjon',
        manufacturer: 'Produsent',
        product: 'Produkt',
        serial: 'Serienummer',
        configuration: 'Konfigurasjonsstreng',
        hidLayout: 'HID-enheter',
        hidRoleKeyboard: 'Tastatur',
        hidRoleRelative: 'Mus (relativ)',
        hidRoleAbsolute: 'Peker (absolutt)',
        hidOff: 'Ikke til stede',
        hidInterface: 'Grensesnitt {{index}}',
        hidBootKeyboardShared:
          'Tastaturet deler et grensesnitt og tilbyr derfor ikke lenger rapport i boot-protokoll. Noen BIOS- og UEFI-oppsett vil ikke se det.',
        functions: 'Funksjoner',
        descriptorAssets: 'Lagrede deskriptorfiler: {{count}}',
        endpointUse:
          'IN {{inUse}} i bruk, {{inFree}} ledige; OUT {{outUse}} i bruk, {{outFree}} ledige',
        apply: 'Bruk',
        applyTitle: 'Vil du bruke denne USB-profilen?',
        applyDesc: 'NanoKVM vil presentere {{profile}} for datamaskinen som er koblet til.',
        reconnect:
          'Tastatur, mus og andre USB-funksjoner kobles fra en kort stund mens gadgeten knyttes til på nytt.',
        applyLinks: 'Knytter til: {{functions}}',
        applyRemoves: 'Fjerner: {{functions}}',
        applyNoHid:
          'Ingen HID-funksjon er igjen etter denne bruken. Tastatur og mus slutter å virke.',
        applyRollback: 'En mislykket bruk går tilbake til {{profile}}.',
        recoveryPowerCycle:
          'Ingen HID overlever denne bruken, så en vert som slutter å svare kan bare reddes ved å slå den av og på.',
        recoveryReboot:
          'Et grensesnitt forsvinner fra den sammensatte enheten; verten kan trenge en omstart for å binde resten på nytt.',
        recoveryHdmiReset:
          'En videofunksjon bygges opp på nytt, så opptakskjeden bak den nullstilles.',
        recoveryReconnect:
          'Verten opplister enheten på nytt; USB-funksjoner faller ut en kort stund.',
        cancel: 'Avbryt',
        noFunctions: 'Ingen tilknyttede funksjoner',
        loadFailed: 'Kunne ikke laste presentasjonsprofilene',
        operationFailed: 'Presentasjonsoperasjonen mislyktes'
      },
      passthrough: {
        title: 'USB-gjennomgang',
        loading: 'Laster...',
        mode: 'Modus',
        hybrid: 'Hybrid',
        exact: 'Nøyaktig',
        hybridDesc: 'Beholder boot-tastaturet og den relative musen, for kompatible enheter.',
        exactDesc: 'Erstatter hver eneste USB-funksjon på NanoKVM med den importerte enheten.',
        hybridWarning: 'Hybrid holder tastaturet og den relative musen tilgjengelige',
        hybridWarningDesc:
          'Lagring, USB-nettverk og den absolutte pekeren kobles fra mens den importerte funksjonen er aktiv.',
        hidWarning: 'Å starte gjennomgang gir fra seg tastaturet, musen og virtuelle medier',
        hidWarningDesc:
          'NanoKVM har bare én USB-enhetskontroller, og proxyen trenger hele. Mens en økt kjører, ser derfor den eksterne verten den videresendte enheten i stedet for NanoKVMs tastatur, mus og virtuelle medier. De kommer tilbake av seg selv i det øyeblikket økten stoppes. Dette nettgrensesnittet påvirkes ikke, så du kan alltid stoppe en økt fra denne siden.',
        hidWarningSafeDesc:
          'NanoKVM har bare én USB-enhetskontroller, og proxyen trenger hele. Mens en økt kjører, ser derfor den eksterne verten den videresendte enheten i stedet for NanoKVMs tastatur, mus og virtuelle medier. De kommer tilbake når økten stoppes.',
        isoLabel: 'Tillat isokrone overføringer',
        isoHint:
          'Slipper gjennom nettkameraer, mikrofoner og andre strømmeenheter. Ingen har målt hva denne maskinvaren klarer.',
        isoWarning:
          'Isokron strømming er uprøvd her og kan holde på tastaturet og musen til du stopper økten',
        info: {
          title: 'Info',
          hybrid:
            'Hybridmodus holder tastaturet og den relative musen tilgjengelige. Lagring, USB-nettverk og den absolutte pekeren kobles fra mens den importerte enheten er aktiv.',
          exact:
            'Nøyaktig modus erstatter hver eneste USB-funksjon på NanoKVM med den importerte enheten. Tastaturet, musen og de virtuelle mediene kommer tilbake av seg selv når økten stoppes.',
          udc: 'NanoKVM har bare én USB-enhetskontroller, og proxyen trenger hele — derfor forsvinner funksjonene ovenfor så lenge en økt varer.',
          web: 'Dette nettgrensesnittet påvirkes ikke, så du kan alltid stoppe en økt fra denne siden.',
          network:
            'Start gjennomgang over Ethernet eller Wi-Fi. Å starte den fra NanoKVMs USB-nettverk avvises, fordi den forbindelsen ville forsvinne.',
          iso: 'Nettkameraer, mikrofoner og andre isokrone enheter avvises til du tillater isokrone overføringer. Den veien virker, men er aldri målt på denne maskinvaren, så regn gjennomstrømningen som ukjent.',
          camera:
            'Nettleserens kamera og mikrofon under Enheter er fortsatt den utprøvde måten å gi verten et på.'
        },
        session: 'Økt',
        activeDesc: 'En enhet er importert, og proxyen holder USB-kontrolleren.',
        inactiveDesc: 'Ingen økt kjører. Tastatur, mus og virtuelle medier fungerer som normalt.',
        device: 'Enhet',
        busId: 'Buss-ID',
        speed: 'Hastighet',
        exporter: 'Eksportør',
        local: 'Importert som',
        localValue: 'Buss {{bus}}, adresse {{address}}',
        udc: 'USB-kontroller',
        pid: 'Proxy-PID',
        startedAt: 'Startet',
        isoDevice:
          'Denne enheten strømmer over isokrone endepunkter, noe som aldri er målt på denne maskinvaren',
        exporterLabel: 'Adressen til eksportøren',
        exporterHint:
          'Verten og porten NanoKVM ringer opp. Gjennom tunnelen nedenfor er det {{exporter}}.',
        busIdLabel: 'Buss-ID på din egen maskin',
        busIdHint: 'Busid-en som usbip list -l viser for enheten, for eksempel {{example}}.',
        start: 'Start gjennomgang',
        stop: 'Stopp gjennomgang',
        startTitle: 'Vil du starte USB-gjennomgang?',
        startDevice: 'NanoKVM vil importere {{busId}} fra {{exporter}}.',
        startHid:
          'USB-tastaturet, musen og virtuelle medier slutter å virke så lenge økten varer, og starter av seg selv igjen når du stopper den.',
        startIso:
          'Nettkameraer og andre isokrone enheter krever at du slår på den isokrone bryteren før du starter.',
        startWeb:
          'Dette nettgrensesnittet fortsetter å virke, så du kan stoppe økten fra denne siden når som helst.',
        startNetwork:
          'Bruk denne siden over Ethernet eller Wi-Fi. Å starte fra NanoKVMs USB-nettverk avvises fordi den forbindelsen ville forsvinne.',
        okBtn: 'Start',
        cancelBtn: 'Avbryt',
        instructions: 'På din egen maskin',
        instructionsDesc:
          'Det finnes bevisst ingen klientagent å installere. Kjør disse vanlige usbip-kommandoene på maskinen enheten er koblet til.',
        copyFailed: 'Kopiering mislyktes. Kopier kommandoen manuelt.',
        copyInsecure:
          'Denne siden leveres ikke over HTTPS, så nettleseren blokkerer kopiering. Kopier kommandoen manuelt, eller slå på HTTPS under Innstillinger, Nettverk.',
        directNote:
          'Uten tunnel må usbipd være tilgjengelig på nettverket ditt, og eksportøradressen ovenfor må peke på den. usbip sender enheten ukryptert, så tunnelen er å foretrekke.',
        steps: {
          modprobe: {
            title: 'Last inn driveren på eksportsiden',
            desc: 'usbip-host er det som lar kjernen gi fra seg en lokal enhet. Den lastes ikke inn som standard.'
          },
          list: {
            title: 'Finn enheten',
            desc: 'Viser hver lokale enhet med busid og leverandør:produkt-paret. Noter busid-en til den du vil bruke.'
          },
          bind: {
            title: 'Bind den til usbip',
            desc: 'Tar enheten fra den vanlige driveren, så den slutter å virke på denne maskinen til du løser bindingen.'
          },
          serve: {
            title: 'Gjør den tilgjengelig',
            desc: 'usbipd blir stående i forgrunnen og venter på at NanoKVM importerer enheten.',
            notice:
              'Vanlig usbipd har ingen valgmulighet for lytteadresse og lytter på alle grensesnitt. Hold port {{port}} stengt i brannmuren, og la bare tunnelen nedenfor nå den.'
          },
          tunnel: {
            title: 'Pek den mot NanoKVM',
            desc: 'En omvendt SSH-tunnel: port {{port}} på NanoKVMs egen loopback blir usbipd på denne maskinen. La den kjøre gjennom hele økten.'
          },
          exporter: {
            title: 'Bruk dette som eksportør',
            desc: 'Skriv dette i eksportørfeltet ovenfor, oppgi buss-ID-en og start økten.'
          },
          unbind: {
            title: 'Gi enheten tilbake',
            desc: 'Etter at økten er stoppet, gir dette enheten tilbake til den vanlige driveren på denne maskinen.'
          }
        }
      },
      mcp: {
        title: 'MCP-tjeneste',
        service: 'MCP-fjernstyring',
        serviceDesc: 'La klarerte MCP-klienter styre tastatur og mus og ta skjermbilder',
        securityWarning:
          'Alle med denne API-nøkkelen kan styre den eksterne verten og se skjermen. Bruk HTTPS, og aktiver tjenesten bare på klarerte nettverk.',
        endpoint: 'Endepunkt',
        apiKey: 'API-nøkkel',
        regenerateConfirmTitle: 'Generere MCP API-nøkkelen på nytt?',
        regenerateConfirmDesc: 'Den gjeldende nøkkelen slutter å virke umiddelbart.',
        enableConfirmTitle: 'Aktivere ekstern MCP-styring?',
        enableConfirmDesc:
          'Aktivering av MCP stopper PicoClaw og lukker alle aktive PicoClaw-økter.',
        failed: 'MCP-operasjonen mislyktes',
        copyFailed: 'Kopiering mislyktes. Kopier manuelt.',
        copyInsecure:
          'Denne siden leveres ikke over HTTPS, så nettleseren blokkerer kopiering. Kopier manuelt, eller slå på HTTPS under Innstillinger, Nettverk.',
        okBtn: 'Bekreft',
        cancelBtn: 'Avbryt'
      },
      about: {
        title: 'Om NanoKVM',
        information: 'Informasjon',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applikasjonsversjon',
        applicationTip: 'Versjon av NanoKVM-webapplikasjonen',
        image: 'Arkivfil-versjon',
        imageTip: 'Versjon av NanoKVM-systemavbildningen',
        deviceKey: 'Enhetsnøkkel',
        community: 'Fellesskap',
        hostname: 'Vertsnavn',
        hostnameUpdated: 'Vertsnavn oppdatert. Start på nytt for å søke.',
        ipType: {
          Wired: 'Kablet',
          Wireless: 'Trådløs',
          Other: 'Annet'
        }
      },
      appearance: {
        title: 'Utseende',
        display: 'Skjerm',
        language: 'Språk',
        languageDesc: 'Velg språket for grensesnittet',
        webTitle: 'Netttittel',
        webTitleDesc: 'Tilpass nettsidetittelen',
        favicon: 'Favicon',
        faviconDesc: 'Tilpass ikonet i nettleserfanen',
        faviconPlaceholder: 'Bilde-URL',
        faviconUpload: 'Last opp',
        faviconReset: 'Tilbakestill',
        faviconCustom: 'Egendefinert ikon',
        faviconBoot: 'Ikon fra /boot/logo.ico',
        faviconDefault: 'Standardikon',
        faviconOverridesBoot: 'Overstyrer /boot/logo.ico',
        faviconErrUrl: 'Skriv inn en http:// eller https:// bildeadresse',
        faviconErrFetch: 'Enheten kunne ikke laste ned bildet',
        faviconErrLarge: 'Bildet er for stort. Grensen er 256 KB',
        faviconErrType: 'Bildeformatet støttes ikke. Bruk .ico, .png, .svg, .gif eller .jpg',
        faviconErrSave: 'Kunne ikke lagre ikonet',
        menuBar: {
          title: 'Menylinje',
          mode: 'Visningsmodus',
          modeDesc: 'Vis menylinje på skjermen',
          modeOff: 'Av',
          modeAuto: 'Skjul automatisk',
          modeAlways: 'Alltid synlig',
          keyboardLedStatus: 'Indikatorer for tastaturlås',
          keyboardLedStatusDesc:
            'Vis Num Lock-, Caps Lock- og Scroll Lock-status for den eksterne datamaskinen',
          icons: 'Undermenyikoner',
          iconsDesc: 'Vis undermenyikoner i menylinjen'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Status for låser på eksternt tastatur',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'På',
        off: 'Av',
        unknown: 'Ukjent'
      },
      device: {
        title: 'Enhet',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Aldri',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 time'
        },
        ssh: {
          description: 'Aktiver SSH ekstern tilgang',
          tip: 'Angi et sterkt passord før du aktiverer (Konto - Endre passord)'
        },
        advanced: 'Avanserte innstillinger',
        swap: {
          title: 'Bytt',
          disable: 'Deaktiver',
          description: 'Angi størrelsen på byttefilen',
          tip: 'Aktivering av denne funksjonen kan forkorte SD-kortets brukbare levetid!'
        },
        mouseJiggler: {
          title: 'Mus Jiggler',
          description: 'Hindre den eksterne verten fra å sove',
          disable: 'Deaktiver',
          absolute: 'Absolutt modus',
          relative: 'Relativ modus'
        },
        mdns: {
          description: 'Aktiver mDNS oppdagelsestjeneste',
          tip: 'Slå den av hvis den ikke er nødvendig'
        },
        hdmi: {
          description: 'Aktiver HDMI/skjermutgang',
          idleTimeoutTitle: 'Tidsavbrudd for inaktivt opptak',
          idleTimeoutDescription: 'Stopp HDMI-opptak etter at det ikke har vært aktive seere i',
          minutes: 'min'
        },
        autostart: {
          title: 'Autostart skriptinnstillinger',
          description: 'Administrer skript som kjører automatisk ved systemstart',
          new: 'Ny',
          deleteConfirm: 'Er du sikker på at du vil slette denne filen?',
          yes: 'Ja',
          no: 'Nei',
          scriptName: 'Autostart skriptnavn',
          scriptContent: 'Autostart skriptinnhold',
          settings: 'Innstillinger'
        },
        hidOnly: 'HID-Bare modus',
        hidOnlyDesc: 'Slutt å emulere virtuelle enheter, behold bare grunnleggende HID-kontroll',
        disk: 'Virtuell disk',
        diskDesc: 'Mount virtual U-disk on the remote host',
        rebindNotice:
          'Å slå om denne bryteren teller opp USB-enheten på nytt, så målet mister kort de virtuelle enhetene og USB-nettverket sitt.',
        media: {
          title: 'Kamera-, mikrofon- og høyttalerplasser',
          desc: 'Angi hvilke medieenheter vertsmaskinen ser. Endepunktsbudsjettet kontrolleres når USB-profilen tas i bruk. Lagring opplister enheten på nytt og kobler fra tilkoblede nettlesere.',
          cameras: 'Kameraer',
          microphones: 'Mikrofoner',
          speakers: 'Høyttalere',
          name: 'Navn',
          namePlaceholder: 'Vises på vertsmaskinen',
          addCamera: 'Legg til kamera',
          addMicrophone: 'Legg til mikrofon',
          addSpeaker: 'Legg til høyttaler',
          remove: 'Fjern',
          cameraDefault: 'Kamera {{index}}',
          microphoneDefault: 'Mikrofon {{index}}',
          speakerDefault: 'Høyttaler {{index}}',
          nameRequired: 'Hver plass trenger et navn.',
          budgetHint:
            'De seks USB IN-endepunktene er en fast maskinvaregrense. Samle tastatur, mus og absolutt pekeenhet på ett HID-grensesnitt under USB-presentasjon, eller slå av den virtuelle disken her eller USB-nettverkskortet under Nettverk.',
          unsupported:
            'Denne kjernen kan ikke navngi medieenheter, så verter viser standardnavnet.',
          save: 'Lagre plasser',
          disconnect: 'Koble fra',
          disconnectAll: 'Koble fra alle kilder',
          limit: 'Kamera-, mikrofon- og høyttalerplasser kan til sammen være høyst åtte.',
          failed: 'Medieplassene kunne ikke oppdateres.'
        },
        reboot: 'Start på nytt',
        rebootDesc: 'Er du sikker på at du vil starte NanoKVM på nytt?',
        okBtn: 'Ja',
        cancelBtn: 'Nei'
      },
      network: {
        title: 'Nettverk',
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurer Wi-Fi',
          apMode: 'AP-modus er aktivert, koble til Wi-Fi ved å skanne QR-koden',
          connect: 'Koble til Wi-Fi',
          connectDesc1: 'Skriv inn nettverkets SSID og passord',
          connectDesc2: 'Skriv inn passordet for å koble til dette nettverket',
          disconnect: 'Er du sikker på at du vil koble fra nettverket?',
          failed: 'Tilkobling mislyktes, prøv igjen.',
          ssid: 'Navn',
          password: 'Passord',
          joinBtn: 'Koble til',
          confirmBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        tls: {
          description: 'Aktiver HTTPS-protokoll',
          tip: 'Merk: Bruk av HTTPS kan øke forsinkelsen, spesielt i MJPEG-videomodus.'
        },
        usb: {
          title: 'USB-nettverkskort',
          desc: 'Gir den styrte datamaskinen et nettverkskort over USB',
          type: 'Korttype',
          typeDesc: 'NCM for moderne systemer, RNDIS for eldre Windows'
        },
        bridge: {
          title: 'Kortet er koblet til',
          lan: 'Nettverket ditt',
          kvmOnly: 'Bare NanoKVM',
          lanDesc:
            'Datamaskinen kommer inn på nettverket ditt gjennom Ethernet-porten på NanoKVM, med sin egen adresse fra ruteren.',
          kvmOnlyDesc:
            'Datamaskinen får adressen sin fra NanoKVM og når NanoKVM, men ingenting utenfor.',
          loading: 'Laster...',
          state: 'Status',
          states: {
            disabled: 'Bare NanoKVM',
            enabled: 'Nettverket ditt',
            rolledBack: 'Rullet tilbake',
            failed: 'Mislyktes',
            pending: 'Pågår'
          },
          uplink: 'Opplenke',
          ports: 'Porter',
          up: 'aktiv',
          down: 'inaktiv',
          noLink: 'ingen link',
          enableTitle: 'Koble datamaskinen til nettverket ditt?',
          disableTitle: 'Begrense datamaskinen til bare NanoKVM?',
          reconnect:
            'Administrasjonstilkoblingen brytes kort og kobles til igjen mens adressen flyttes.',
          rollback:
            'Hvis verifiseringen mislykkes, gjenopprettes den forrige konfigurasjonen automatisk.',
          enableBtn: 'Koble til nettverket mitt',
          disableBtn: 'Bare NanoKVM',
          cancelBtn: 'Avbryt',
          interrupted: 'Tilkoblingen ble brutt under bruken. Sjekker gjeldende status på nytt.',
          pendingNotice:
            'En endring av broen pågår fortsatt eller ble avbrutt før den ble fullført.',
          revert: 'Gjenopprett forrige konfigurasjon',
          rolledBackNotice:
            'Den siste endringen ble rullet tilbake, og den forrige konfigurasjonen ble gjenopprettet.',
          verifyFailed: 'Verifiseringen mislyktes: {{gates}}',
          gates: {
            address: 'adresse',
            gateway: 'gateway',
            inbound: 'innkommende tilkobling'
          },
          inboundWeak:
            'Kontrollen av innkommende trafikk gikk gjennom bare fordi NanoKVM koblet til seg selv. Det viser at webtjenesten lytter og er tilgjengelig lokalt, ikke at en forespørsel fra nettverket kommer frem.',
          noCarrier:
            'Ingen link på {{port}}. Broen har ingen vei ut på nettverket før det kobles til en kabel.',
          loop: 'Ruteren læres også på {{port}}, så den porten er en annen vei til det samme nettverket. Spanning tree er av, så ingenting her bryter sløyfen: koble fra en av de to veiene.',
          failedNotice:
            'Den siste endringen kunne ikke angres. NanoKVM er kanskje bare tilgjengelig via Wi-Fi-tilgangspunktet eller en seriell konsoll.'
        },
        dns: {
          title: 'DNS',
          description: 'Konfigurer DNS-servere for NanoKVM',
          mode: 'Modus',
          dhcp: 'DHCP',
          manual: 'Manuell',
          add: 'Legg til DNS',
          save: 'Lagre',
          invalid: 'Skriv inn en gyldig IP-adresse',
          noDhcp: 'Ingen DHCP-DNS er tilgjengelig nå',
          saved: 'DNS-innstillinger lagret',
          saveFailed: 'Kunne ikke lagre DNS-innstillinger',
          unsaved: 'Ulagrede endringer',
          maxServers: 'Maksimalt {{count}} DNS-servere er tillatt',
          dnsServers: 'DNS-servere',
          dhcpServersDescription: 'DNS-servere hentes automatisk fra DHCP',
          manualServersDescription: 'DNS-servere kan redigeres manuelt',
          networkDetails: 'Nettverksdetaljer',
          interface: 'Grensesnitt',
          ipAddress: 'IP-adresse',
          subnetMask: 'Subnettmaske',
          router: 'Ruter',
          none: 'Ingen'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC-tjener',
        description:
          'La enhver VNC-klient se den eksterne skjermen og bruke tastatur og mus, med innlogging via NanoKVM-kontoen din',
        port: 'Port',
        portDescription: 'Koble til denne porten på NanoKVM-adressen'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Minneoptimalisering',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Bytt minne',
          tip: 'Hvis problemene vedvarer etter at du har aktivert minneoptimalisering, prøv å aktivere swap-minne. Dette setter swap-filstørrelsen til 256MB som standard, som kan justeres i "Innstillinger > Enhet".'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Laster...',
        notInstall: 'Tailscale er ikke funnet! Vennligst installer.',
        install: 'Installér',
        installing: 'Installerer',
        failed: 'Installering feilet',
        retry: 'Vennligst last inn siden på nytt og forsøk igjen eller installer manuelt',
        download: 'Last ned',
        package: 'installasjonspakken',
        unzip: 'og pakk den ut',
        upTailscale: 'Last opp Tailscale til NanoKVM-enhetens mappe /usr/bin/',
        upTailscaled: 'Last opp tailscaled til NanoKVM-enhetens mappe /usr/sbin/',
        refresh: 'Last inn denne siden på nytt',
        notRunning: 'Tailscale kjører ikke. Start den for å fortsette.',
        run: 'Start',
        notLogin:
          'Denne enheten er ikke knyttet til din konto enda. Vennligst logg inn og knytt den til kontoen din..',
        urlPeriod: 'Denne lenken er gyldig i 10 minutter',
        login: 'Logg inn',
        loginSuccess: 'Logget inn',
        enable: 'Skru på Tailscale',
        deviceName: 'Enhetens navn',
        deviceIP: 'Enhetens IP',
        account: 'Konto',
        logout: 'Logg ut',
        logoutDesc: 'Er du sikker på at du vil logge ut?',
        uninstall: 'Avinstaller Tailscale',
        uninstallDesc: 'Er du sikker på at du vil avinstallere Tailscale?',
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
        loading: 'Laster...',
        notInstall: 'Ikke installert',
        notConfigured: 'Ikke konfigurert',
        stopped: 'Stoppet',
        running: 'Kjører',
        connected: 'Tilkoblet',
        error: 'Feil',
        atBoot: 'starter ved oppstart',
        notAtBoot: 'starter ikke ved oppstart',
        arguments: 'Argumenter',
        argumentsTip: 'Kommandolinjeargumenter som sendes til tjenesten ved oppstart.',
        env: 'Miljøvariabler',
        envKey: 'Navn',
        envValue: 'Verdi',
        envAdd: 'Legg til variabel',
        envRemove: 'Fjern',
        configured: 'Konfigurert',
        save: 'Lagre',
        saved: 'Konfigurasjonen er lagret',
        start: 'Start',
        stop: 'Stopp',
        restart: 'Start på nytt',
        logs: 'Logg',
        logsEmpty: 'Ingen loggoppføringer ennå',
        refresh: 'Oppdater',
        binary: 'Binærfil',
        binaryShipped: 'Følger med fastvaren',
        binaryCustom: 'Egen binærfil',
        binaryUpload: 'Last opp binærfil',
        binaryRevert: 'Gjenopprett medfølgende binærfil',
        binaryRevertDesc:
          'Slette den opplastede binærfilen og gjenopprette den som følger med fastvaren?',
        serverWarning: 'En server uten begrensninger fungerer som en åpen proxy',
        noHealthSignal:
          'Denne tjenesten rapporterer ingen helsestatus, så NanoKVM vet bare at prosessen kjører, ikke om tunnelen er tilkoblet.',
        memoryWarning: 'Å kjøre flere fjerntilgangstjenester samtidig kan tømme minnet',
        resources: 'Ressurser',
        memory: {
          title: 'Minnegrense',
          description:
            'Begrenser Go-heapen til newt til {{limit}} MiB fra neste omstart. Dens egen grense, ikke Tailscales; slått av gjelder Gos standard, og GOGC=50 brukes uansett.',
          noRuntime:
            'wstunnel er Rust: ingen søppelsamler og ingen heap-grense å sette, og arbeidstrådene følger allerede den ene CPU-en på enheten.',
          notApplicable: 'Ikke aktuelt'
        },
        swap: {
          title: 'Vekslefil',
          description:
            'Legger til en vekslefil på 256 MB på SD-kortet. Den gjelder hele systemet: samme veksling betjener Tailscale, KVM-tjeneren og alt annet på enheten.'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nei'
      },
      update: {
        title: 'Se etter oppdatering',
        queryFailed: 'Kunne ikke hente versjon',
        updateFailed: 'En feil oppstod under oppdatering. Vennligst forsøk igjen.',
        isLatest: 'Du har siste versjon allerede.',
        rebooting:
          'Installerer den nye kjernen og starter på nytt. Dette kan ta noen minutter; ikke slå av strømmen.',
        kernelUpdate:
          'Denne oppdateringen installerer kjerne {{version}}. Enheten starter på nytt og går selv tilbake til den nåværende kjernen hvis den nye ikke starter.',
        rolledBack: 'Kjerne {{version}} startet ikke, og enheten gikk tilbake til forrige kjerne.',
        available: 'En oppdatering er tilgjengelig. Er du sikker på at du ønsker å oppdatere?',
        updating: 'Oppdatering har startet. Vennligst vent...',
        confirm: 'Oppdater',
        cancel: 'Avbryt',
        preview: 'Forhåndsvisningsoppdateringer',
        previewDesc: 'Få tidlig tilgang til nye funksjoner og forbedringer',
        previewTip:
          'Vær oppmerksom på at forhåndsvisningsutgivelser kan inneholde feil eller ufullstendig funksjonalitet!',
        customServer: {
          title: 'Egendefinert oppdateringsserver',
          desc: 'Se etter og last ned nettbaserte oppdateringer fra en angitt server',
          invalidUrl:
            'Angi en gyldig HTTP- eller HTTPS-servermappe uten spørring, fragment eller latest.json.',
          loadFailed: 'Kunne ikke laste inn konfigurasjonen for oppdateringsserveren.',
          saveFailed: 'Kunne ikke lagre konfigurasjonen for oppdateringsserveren.',
          saved: 'Konfigurasjonen for oppdateringsserveren er lagret.',
          save: 'Lagre',
          confirmTitle: 'Vil du bruke en egendefinert oppdateringsserver?',
          confirmDesc:
            'SHA-512 kontrollerer bare at pakken samsvarer med manifestet fra denne serveren. Det beviser ikke at pakken er en offisiell NanoKVM-utgivelse. En feilkonfigurert eller ondsinnet server kan gjøre enheten ubrukelig, føre til tap av data eller kompromittere systemet.',
          confirm: 'Bruk likevel',
          previewDisabled:
            'Forhåndsvisningsoppdateringer er ikke tilgjengelige mens en egendefinert oppdateringsserver er aktivert.'
        },
        offline: {
          kernelNotice:
            'Denne pakken inneholder en kjerne. Den skrives til reserveplassen og enheten starter på nytt for å prøve den; kommer den ikke tilbake, går enheten selv tilbake til den nåværende kjernen.',
          kernelConfirm: 'Installer kjerne',
          kernelCancel: 'Avbryt',
          title: 'Offline oppdateringer',
          desc: 'Oppdater gjennom lokal installasjonspakke',
          upload: 'Last opp',
          checksumPlaceholder: 'SHA-256-sjekksum (valgfritt)',
          invalidChecksum: 'SHA-256-sjekksummen må inneholde 64 heksadesimale tegn.',
          checksumMismatch: 'SHA-256-verifiseringen mislyktes. Pakken kan være skadet.',
          invalidName: 'Ugyldig filnavnformat. Last ned fra GitHub-utgivelser.',
          updateFailed: 'En feil oppstod under oppdatering. Vennligst forsøk igjen.'
        }
      },
      account: {
        title: 'Konto',
        webAccount: 'Navn på webkonto',
        role: 'Rolle',
        roles: {
          admin: 'Administrator',
          user: 'Bruker'
        },
        password: 'Passord',
        updateBtn: 'Update',
        logoutBtn: 'Logg ut',
        logoutDesc: 'Er du sikker på at du vil logge ut?',
        okBtn: 'Ja',
        cancelBtn: 'Nei',
        users: {
          title: 'Brukere',
          create: 'Opprett bruker',
          enabled: 'Aktivert',
          disabled: 'Deaktivert',
          deviceOwner: 'Enhetens eier',
          resetPassword: 'Tilbakestill passord',
          delete: 'Slett',
          deleteConfirm: 'Slette denne brukeren og trekke tilbake alle øktene til vedkommende?',
          created: 'Bruker opprettet',
          deleted: 'Bruker slettet',
          passwordUpdated: 'Passord oppdatert',
          loadFailed: 'Kunne ikke laste brukere',
          saveFailed: 'Kunne ikke lagre brukeren',
          deleteFailed: 'Kunne ikke slette brukeren'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistent',
      empty: 'Åpne panelet og start en oppgave for å begynne.',
      inputPlaceholder: 'Beskriv hva du vil at PicoClaw skal gjøre',
      newConversation: 'Ny samtale',
      processing: 'Behandler...',
      agent: {
        defaultTitle: 'Generell assistent',
        defaultDescription: 'Generell chat-, søk- og arbeidsområdehjelp.',
        kvmTitle: 'Fjernstyring',
        kvmDescription: 'Betjen den eksterne verten gjennom NanoKVM.',
        switched: 'Agentrolle byttet',
        switchFailed: 'Kunne ikke bytte agentrolle'
      },
      send: 'Send',
      cancel: 'Avbryt',
      status: {
        connecting: 'Kobler til gateway...',
        connected: 'PicoClaw-økt tilkoblet',
        disconnected: 'PicoClaw-økt frakoblet',
        stopped: 'Stoppforespørsel sendt',
        runtimeStarted: 'PicoClaw runtime startet',
        runtimeStartFailed: 'Kunne ikke starte PicoClaw runtime',
        runtimeStopped: 'PicoClaw runtime stoppet',
        runtimeStopFailed: 'Kunne ikke stoppe PicoClaw runtime',
        controlSwitchedToMCP: 'Styringen er byttet til den eksterne MCP-tjenesten'
      },
      connection: {
        runtime: {
          checking: 'Kontrollerer',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime klar',
          stopped: 'Runtime stoppet',
          blockedByMCP: 'Ekstern MCP-styring er aktiv',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime utilgjengelig',
          configError: 'Konfigurasjonsfeil'
        },
        transport: {
          connecting: 'Kobler til',
          connected: 'Tilkoblet',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Inaktiv',
          busy: 'Opptatt'
        }
      },
      message: {
        toolAction: 'Handling',
        observation: 'Observasjon',
        screenshot: 'Skjermbilde'
      },
      overlay: {
        locked: 'PicoClaw kontrollerer enheten. Manuell inntasting er satt på pause.'
      },
      control: {
        picoclaw: 'Enhetsstyring: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Enhetsstyring: ekstern MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Enhetsstyring: av',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Gi styring',
        release: 'Frigi',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw-styring gitt',
        released: 'PicoClaw-styring frigitt',
        grantFailed: 'Kunne ikke gi PicoClaw styring',
        releaseFailed: 'Kunne ikke frigi PicoClaw styring',
        grantConfirmTitle: 'Bytte enhetsstyring til PicoClaw?',
        grantConfirmDesc: 'Eksterne MCP-enhetsskrivinger blir avbrutt.'
      },
      install: {
        install: 'Installer PicoClaw',
        installing: 'Installerer PicoClaw',
        success: 'PicoClaw installert vellykket',
        failed: 'Kunne ikke installere PicoClaw',
        uninstalling: 'Avinstallerer runtime...',
        uninstalled: 'Runtime ble avinstallert.',
        uninstallFailed: 'Avinstallering mislyktes.',
        requiredTitle: 'PicoClaw er ikke installert',
        requiredDescription: 'Installer PicoClaw før du starter PicoClaw runtime.',
        progressDescription: 'PicoClaw blir lastet ned og installert.',
        stages: {
          preparing: 'Forbereder',
          downloading: 'Laster ned',
          extracting: 'Pakker ut',
          verifying: 'Verifiserer',
          installing: 'Installerer',
          installed: 'Installert',
          install_timeout: 'Tidsavbrudd',
          install_failed: 'Mislyktes'
        }
      },
      model: {
        requiredTitle: 'Modellkonfigurasjon er nødvendig',
        requiredDescription: 'Konfigurer PicoClaw-modellen før du bruker PicoClaw chat.',
        docsTitle: 'Konfigurasjonsveiledning',
        docsDesc: 'Støttede modeller og protokoller',
        menuLabel: 'Konfigurer modell',
        modelIdentifier: 'Modellidentifikator',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API-nøkkel',
        apiKeyPlaceholder: 'Skriv inn modellens API-nøkkel',
        save: 'Lagre',
        saving: 'Lagrer',
        saved: 'Modellkonfigurasjon lagret',
        saveFailed: 'Kunne ikke lagre modellkonfigurasjonen',
        invalid: 'Modellidentifikator, API Base URL og API-nøkkel kreves'
      },
      uninstall: {
        menuLabel: 'Avinstaller',
        confirmTitle: 'Avinstaller PicoClaw',
        confirmContent:
          'Er du sikker på at du vil avinstallere PicoClaw? Dette vil slette den kjørbare filen og alle konfigurasjonsfilene.',
        confirmOk: 'Avinstaller',
        confirmCancel: 'Avbryt'
      },
      history: {
        title: 'Historikk',
        loading: 'Laster inn økter...',
        emptyTitle: 'Ingen historikk ennå',
        emptyDescription: 'Tidligere PicoClaw økter vil vises her.',
        loadFailed: 'Kunne ikke laste inn økthistorikk',
        deleteFailed: 'Kunne ikke slette økten',
        deleteConfirmTitle: 'Slett økt',
        deleteConfirmContent: 'Er du sikker på at du vil slette "{{title}}"?',
        deleteConfirmOk: 'Slett',
        deleteConfirmCancel: 'Avbryt',
        messageCount_one: '{{count}} melding',
        messageCount_other: '{{count}} meldinger',
        messageCount: '{{count}} meldinger'
      },
      config: {
        startRuntime: 'Start PicoClaw',
        stopRuntime: 'Stopp PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Bytte styringen til PicoClaw?',
        enableConfirmDesc: 'Når PicoClaw startes, deaktiveres den eksterne MCP-tjenesten.',
        enableConfirmOk: 'Start PicoClaw',
        enableConfirmCancel: 'Avbryt',
        title: 'Start PicoClaw',
        description: 'Start runtime for å begynne å bruke PicoClaw-assistenten.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Vi har hatt et problem',
      refresh: 'Oppdater',
      panel: 'Dette panelet sluttet å virke',
      retry: 'Prøv igjen',
      reload: 'Last inn siden på nytt'
    },
    fullscreen: {
      toggle: 'Veksle fullskjerm'
    },
    menu: {
      collapse: 'Skjul meny',
      expand: 'Utvid menyen'
    }
  }
};

export default nb;
