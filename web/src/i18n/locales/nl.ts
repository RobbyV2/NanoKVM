const nl = {
  translation: {
    head: {
      desktop: 'Extern bureaublad',
      login: 'Inloggen',
      changePassword: 'Wachtwoord wijzigen',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Inloggen',
      placeholderUsername: 'Voer gebruikersnaam in',
      placeholderPassword: 'Voer wachtwoord in',
      placeholderCurrentPassword: 'Huidig wachtwoord',
      placeholderPassword2: 'Voer wachtwoord nogmaals in',
      noEmptyUsername: 'Gebruikersnaam mag niet leeg zijn',
      noEmptyPassword: 'Wachtwoord mag niet leeg zijn',
      passwordLength: 'Het wachtwoord moet tussen 8 en 72 tekens lang zijn',
      noAccount:
        'Ophalen van gebruikersinformatie mislukt, vernieuw de webpagina of reset het wachtwoord',
      invalidUser: 'Ongeldige gebruikersnaam of wachtwoord',
      locked: 'Te veel aanmeldingen, probeer het later opnieuw',
      globalLocked: 'Systeem wordt beveiligd. Probeer het later opnieuw',
      error: 'Onverwachte fout',
      invalidCurrentPassword: 'Het huidige wachtwoord is onjuist',
      changePassword: 'Wachtwoord wijzigen',
      changePasswordDesc:
        'Voor de veiligheid van uw apparaat, wijzig alstublieft het webaanmeldingswachtwoord.',
      differentPassword: 'Wachtwoorden komen niet overeen',
      illegalUsername: 'Gebruikersnaam bevat ongeldige tekens',
      illegalPassword: 'Wachtwoord bevat ongeldige tekens',
      forgetPassword: 'Wachtwoord vergeten',
      ok: 'Ok',
      cancel: 'Annuleren',
      loginButtonText: 'Inloggen',
      tips: {
        reset1:
          'Om de wachtwoorden opnieuw in te stellen, houdt u de BOOT-knop op de NanoKVM 10 seconden lang ingedrukt.',
        reset2: 'Voor gedetailleerde stappen kunt u dit document raadplegen:',
        reset3: 'Standaard webaccount:',
        reset4: 'Standaard SSH-account:',
        change1: 'Houd er rekening mee dat deze actie de volgende wachtwoorden zal wijzigen:',
        change2: 'Web login wachtwoord',
        change3: 'Systeem root-wachtwoord (SSH-inlogwachtwoord)',
        change4:
          'Om de wachtwoorden opnieuw in te stellen, houdt u de BOOT-knop op de NanoKVM ingedrukt.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Wi-Fi configureren voor NanoKVM',
      success: 'Controleer de netwerkstatus van NanoKVM en bezoek het nieuwe IP-adres.',
      failed: 'De bewerking is mislukt. Probeer het opnieuw.',
      invalidMode:
        'De huidige modus ondersteunt geen netwerkconfiguratie. Ga naar uw apparaat en schakel de configuratiemodus Wi-Fi in.',
      confirmBtn: 'Ok',
      finishBtn: 'Gereed',
      ap: {
        authTitle: 'Authenticatie vereist',
        authDescription: 'Voer het wachtwoord AP in om door te gaan',
        authFailed: 'Ongeldig AP wachtwoord',
        passPlaceholder: 'AP wachtwoord',
        verifyBtn: 'Verifieer'
      }
    },
    screen: {
      scale: 'Schaal',
      title: 'Scherm',
      video: 'Videomodus',
      videoDirectTips: 'Schakel HTTPS in "Instellingen > Apparaat" in om deze modus te gebruiken',
      resolution: 'Resolutie',
      auto: 'Automatisch',
      autoTips:
        'Bij bepaalde resoluties kunnen schermverscheuringen of muisverplaatsingen optreden. Overweeg de resolutie van de externe host aan te passen of schakel de automatische modus uit.',
      fps: 'FPS',
      customizeFps: 'Aanpassen',
      quality: 'Kwaliteit',
      qualityLossless: 'Verliesvrij',
      qualityHigh: 'Hoog',
      qualityMedium: 'Gemiddeld',
      qualityLow: 'Laag',
      frameDetect: 'Frame detectie',
      frameDetectTip:
        'Berekent het verschil tussen frames. Stopt met het verzenden van de videostream wanneer er geen veranderingen worden gedetecteerd op het scherm van de externe host.',
      resetHdmi: 'Reset HDMI',
      mixedH264: {
        title: 'H.264-streamconflict',
        description:
          'H.264 Direct en H.264 WebRTC worden tegelijkertijd gebruikt. Dit kan tearing of beschadigde video veroorzaken. Gebruik slechts één H.264-modus.'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC-verbinding mislukt',
        description: 'Controleer de netwerkverbinding of wijzig de videomodus.'
      },
      captureStatus: {
        hdmiError: 'HDMI-schermfout',
        unsupportedResolution: 'De huidige resolutie wordt niet ondersteund',
        retrieving: 'Scherm ophalen...',
        changingResolution: 'Resolutie wisselen...',
        updateFailed: 'Het scherm kan nu niet worden bijgewerkt',
        videoError: 'Fout bij videoweergave',
        noHdmi: 'Geen HDMI-signaal gedetecteerd',
        unavailable: 'Het scherm kan nu niet worden weergegeven'
      }
    },
    keyboard: {
      title: 'Toetsenbord',
      paste: 'Plakken',
      tips: 'Alleen standaard toetsenbordletters en symbolen worden ondersteund',
      placeholder: 'Voer tekst in',
      submit: 'Verzenden',
      virtual: 'Toetsenbord',
      readClipboard: 'Lezen vanaf het Klembord',
      clipboardPermissionDenied:
        'Klembordtoestemming geweigerd. Sta klembordtoegang toe in uw browser.',
      clipboardReadError: 'Kan het klembord niet lezen',
      dropdownEnglish: 'Engels',
      dropdownGerman: 'Duits',
      dropdownFrench: 'Frans',
      dropdownRussian: 'Russisch',
      shortcut: {
        title: 'Snelkoppelingen',
        custom: 'Aangepast',
        capture: 'Klik hier om de snelkoppeling vast te leggen',
        clear: 'Duidelijk',
        save: 'Opslaan',
        captureTips:
          'Voor het vastleggen van systeemtoetsen (zoals de Windows-toets) is toestemming voor volledig scherm vereist.',
        enterFullScreen: 'Schakelen naar volledig scherm.'
      },
      leaderKey: {
        title: 'Leader-toets',
        desc: 'Omzeil browserbeperkingen en stuur systeemsnelkoppelingen rechtstreeks naar de externe host.',
        howToUse: 'Hoe te gebruiken',
        simultaneous: {
          title: 'Gelijktijdige modus',
          desc1: 'Houd de Leader-toets ingedrukt en druk daarna op de sneltoets.',
          desc2: 'Intuïtief, maar kan conflicteren met systeemsnelkoppelingen.'
        },
        sequential: {
          title: 'Sequentiële modus',
          desc1:
            'Druk op de Leader-toets → druk de sneltoets in volgorde in → druk opnieuw op de Leader-toets.',
          desc2: 'Vereist meer stappen, maar vermijdt systeemconflicten volledig.'
        },
        enable: 'Leader-toets inschakelen',
        tip: 'Wanneer deze toets als Leader-toets is toegewezen, werkt hij uitsluitend als sneltoetstrigger en verliest hij zijn standaardgedrag.',
        placeholder: 'Druk op de Leader-toets',
        shiftRight: 'Rechter Shift',
        ctrlRight: 'Rechter Ctrl',
        metaRight: 'Rechter Win',
        submit: 'Verzenden',
        recorder: {
          rec: 'OPN',
          activate: 'Toetsen activeren',
          input: 'Druk op de snelkoppeling...'
        }
      }
    },
    mouse: {
      title: 'Muis',
      cursor: 'Cursorstijl',
      default: 'Standaard cursor',
      pointer: 'Aanwijzer cursor',
      cell: 'Cel cursor',
      text: 'Tekst cursor',
      grab: 'Grijp cursor',
      hide: 'Cursor verbergen',
      mode: 'Muismodus',
      absolute: 'Absolute modus',
      relative: 'Relatieve modus',
      direction: 'Scrollwielrichting',
      scrollUp: 'Scroll naar boven',
      scrollDown: 'Scroll naar beneden',
      speed: 'Scrollwielsnelheid',
      fast: 'Snel',
      slow: 'Langzaam',
      requestPointer:
        'Relatieve modus wordt gebruikt. Klik op het bureaublad om de muisaanwijzer te krijgen.',
      resetHid: 'HID resetten',
      hidOnly: {
        title: 'Alleen HID-modus',
        desc: 'Als uw muis en toetsenbord niet meer reageren en het opnieuw instellen van HID niet helpt, kan er sprake zijn van een compatibiliteitsprobleem tussen de NanoKVM en het apparaat. Probeer de modus HID-Only in te schakelen voor betere compatibiliteit.',
        tip1: 'Als u de modus HID-Only inschakelt, worden de virtuele U-schijf en het virtuele netwerk ontkoppeld',
        tip2: 'In de modus HID-Alleen is beeldmontage uitgeschakeld',
        tip3: 'NanoKVM wordt automatisch opnieuw opgestart na het wisselen van modus',
        enable: 'Schakel de modus HID-Alleen in',
        disable: 'Schakel de modus HID-Alleen uit'
      }
    },
    image: {
      title: 'Afbeeldingen',
      loading: 'Laden...',
      empty: 'Niets gevonden',
      mountMode: 'Montagemodus',
      mountFailed: 'Koppelen mislukt',
      mountDesc:
        'In sommige systemen is het noodzakelijk om de virtuele schijf op de externe host uit te werpen voordat het image wordt gekoppeld.',
      unmountFailed: 'Ontkoppelen mislukt',
      unmountDesc:
        'Op sommige systemen moet u de image handmatig uitwerpen van de externe host voordat u de image ontkoppelt.',
      refresh: 'Vernieuw de afbeeldingenlijst',
      attention: 'Let op',
      deleteConfirm: 'Weet u zeker dat u deze afbeelding wilt verwijderen?',
      okBtn: 'Ja',
      cancelBtn: 'Nee',
      tips: {
        title: 'Hoe te uploaden',
        usb1: 'Verbind de NanoKVM met uw computer via USB.',
        usb2: 'Zorg ervoor dat de virtuele schijf is gekoppeld (Instellingen - Virtuele schijf).',
        usb3: 'Open de virtuele schijf op uw computer en kopieer het imagebestand naar de hoofdmap van de virtuele schijf.',
        scp1: 'Zorg ervoor dat de NanoKVM en uw computer zich in hetzelfde lokale netwerk bevinden.',
        scp2: 'Open een terminal op uw computer en gebruik het SCP-commando om het imagebestand te uploaden naar de /data directory op de NanoKVM.',
        scp3: 'Voorbeeld: scp uw-image-pad root@uw-nanokvm-ip:/data',
        tfCard: 'TF-kaart',
        tf1: 'Deze methode wordt ondersteund op Linux-systemen',
        tf2: 'Haal de TF-kaart uit de NanoKVM (voor de VOLLEDIGE versie, demonteer eerst de behuizing).',
        tf3: 'Plaats de TF-kaart in een kaartlezer en verbind deze met uw computer.',
        tf4: 'Kopieer het imagebestand naar de /data directory op de TF-kaart.',
        tf5: 'Plaats de TF-kaart terug in de NanoKVM.'
      }
    },
    script: {
      title: 'Script',
      upload: 'Uploaden',
      run: 'Uitvoeren',
      runBackground: 'Op achtergrond uitvoeren',
      runFailed: 'Uitvoeren mislukt',
      attention: 'Let op',
      delDesc: 'Weet u zeker dat u dit bestand wilt verwijderen?',
      confirm: 'Ja',
      cancel: 'Nee',
      delete: 'Verwijderen',
      close: 'Sluiten'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM Terminal',
      serial: 'Seriële poort terminal',
      serialPort: 'Seriële poort',
      serialPortPlaceholder: 'Voer de seriële poort in',
      baudrate: 'Baudrate',
      parity: 'Pariteit',
      parityNone: 'Geen',
      parityEven: 'Even',
      parityOdd: 'Oneven',
      flowControl: 'Debietregeling',
      flowControlNone: 'Geen',
      flowControlSoft: 'Software',
      flowControlHard: 'Hardware',
      dataBits: 'Databits',
      stopBits: 'Stopbits',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Commando wordt verzonden...',
      sent: 'Commando verzonden',
      input: 'Voer het MAC-adres in',
      ok: 'Ok'
    },
    download: {
      title: 'Afbeeldingdownloader',
      input: 'Voer een externe afbeelding in URL',
      ok: 'Ok',
      disabled: '/data partitie is RO, dus we kunnen de afbeelding niet downloaden',
      uploadbox: 'Zet het bestand hier neer of klik om te selecteren',
      inputfile: 'Voer het afbeeldingsbestand in',
      NoISO: 'Geen ISO',
      sha256: 'SHA-256 (optioneel)',
      sha256Placeholder: 'Voer een SHA-256-controlesom van 64 tekens in',
      invalidSHA256: 'SHA-256 moet een hexadecimale tekenreeks van 64 tekens zijn',
      failed: 'Download mislukt',
      success: 'Download geslaagd',
      checksumFailed: 'Download mislukt: SHA-256-verificatie mislukt',
      cancel: 'Annuleren',
      cancelFailed: 'Download annuleren mislukt'
    },
    power: {
      title: 'Aan/uit',
      showConfirm: 'Bevestiging',
      showConfirmTip: 'Stroombedieningen vereisen een extra bevestiging',
      reset: 'Resetten',
      power: 'Aan/uit',
      powerShort: 'Aan/uit (kort indrukken)',
      powerLong: 'Aan/uit (lang indrukken)',
      resetConfirm: 'Doorgaan met resetten?',
      powerConfirm: 'Doorgaan met stroomvoorziening?',
      okBtn: 'Ja',
      cancelBtn: 'Nee'
    },
    devices: {
      title: 'Apparaten',
      stale: 'De live status van de apparaten is niet beschikbaar. Bezig met opnieuw verbinden.',
      empty:
        'Er zijn geen camera- of microfoonplaatsen ingesteld. Voeg er een toe bij Instellingen, Apparaat.',
      available: 'Beschikbaar',
      waiting: 'De host wacht op een bron',
      hostOpen: 'Host open',
      hostIdle: 'Host inactief',
      sending: 'Verzendt vanuit deze browser',
      black: 'Zwart beeld',
      silence: 'Digitale stilte',
      resuming: 'Wacht op hervatten',
      stop: 'Delen stoppen',
      disconnect: 'Loskoppelen',
      takeover: 'Overnemen',
      refused: 'In gebruik door {{owner}} vanaf {{source}}',
      connectedSources_one: '{{count}} verbonden bron',
      connectedSources_other: '{{count}} verbonden bronnen',
      connectedSources: '{{count}} verbonden bronnen',
      connection: {
        connecting: 'Verbinden',
        connected: 'Live',
        disconnected: 'Opnieuw verbinden'
      },
      share: {
        camera: 'Camera delen',
        microphone: 'Microfoon delen',
        usbDevice: 'USB delen'
      },
      permission: {
        denied: 'Geblokkeerd in de site-instellingen van je browser',
        prompt: 'Je browser vraagt om toegang',
        insecure:
          'Deze pagina wordt niet via HTTPS geleverd, daarom blokkeert de browser dit apparaat. Schakel HTTPS in bij Instellingen, Netwerk.'
      },
      capture: {
        unsupported: 'Deze browser kan geen audio of video opnemen',
        camera: 'Deze browser kan geen camerabeelden coderen',
        microphone: 'Deze browser kan geen microfoonaudio verwerken'
      },
      mic: {
        mute: 'Dempen',
        unmute: 'Dempen opheffen'
      },
      revoked: {
        released: 'Het delen is gestopt',
        lease_expired: 'De lease verliep voordat deze browser terugkwam',
        admin_disconnect: 'Een beheerder heeft alle bronnen losgekoppeld',
        slot_removed: 'De plek is verwijderd',
        slot_changed: 'De plek is opnieuw geconfigureerd',
        taken_over: 'Een beheerder heeft deze plek overgenomen'
      },
      usb: {
        surrendered: 'USB-passthrough houdt het toetsenbord en de muis vast',
        surrenderedDesc:
          'De externe host ziet het geïmporteerde apparaat in plaats van het toetsenbord, de muis en de virtuele media van NanoKVM. Ze komen terug zodra de sessie stopt.',
        unsupported: 'WebUSB vereist een Chromium-browser',
        insecure:
          'Deze pagina wordt niet via HTTPS geleverd, daarom houdt de browser WebUSB tegen. Schakel HTTPS in bij Instellingen, Netwerk.',
        session: '{{device}} wordt doorgegeven ({{mode}})',
        idle: 'Geen passthrough-sessie',
        mode: {
          hybrid: 'hybride',
          exact: 'exact'
        }
      }
    },
    settings: {
      title: 'Instellingen',
      display: {
        title: 'Beeldscherm',
        loading: 'Laden...',
        active: 'Actieve EDID',
        activeUnknown:
          'NanoKVM heeft sinds het opstarten geen EDID geschreven, dus de identiteit die de host ziet is onbekend.',
        appliedAt: 'Toegepast op {{time}}',
        download: 'Downloaden',
        downloadBackup: 'Vorige downloaden',
        preset: 'Monitorvoorinstelling',
        presetPlaceholder: 'Kies een monitor',
        upload: 'Uploaden',
        selected: 'Geselecteerde EDID',
        errors: 'Fouten',
        warnings: 'Waarschuwingen',
        info: 'Informatie',
        unknownMonitor: 'Onbekende monitor',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Audio',
        audioNo: 'Geen audio',
        extensionBlocks: 'Uitbreidingsblokken: {{blocks}}',
        apply: 'Toepassen',
        applyTitle: 'Deze EDID toepassen?',
        before: 'Huidig',
        after: 'Nieuw',
        hdmiNotice:
          'De videoregistratie stopt tijdens het schrijven van de EDID en start daarna vanzelf weer.',
        powerCycleNotice:
          'Dit apparaat moet fysiek van de stroom worden losgekoppeld en opnieuw aangesloten voordat de nieuwe EDID werkt.',
        powerCycleUnverified:
          'De schrijfactie is niet geverifieerd, dus de videochip houdt wat er nu in staat totdat dit apparaat fysiek van de stroom wordt losgekoppeld en weer aangesloten.',
        applied: 'EDID toegepast en geverifieerd.',
        applyFailed: 'Het toepassen van de EDID is mislukt.',
        busy: 'De videochip was bezet. Probeer het opnieuw.',
        unsupported: 'Dit apparaat ondersteunt het wijzigen van de EDID niet.',
        toolMissing: 'Het EDID-hulpprogramma ontbreekt in deze firmware.',
        noAudio:
          'Deze EDID meldt geen audio, dus de host stopt mogelijk met het verzenden van geluid.',
        oldVersion: 'Deze EDID gebruikt een oudere versie dan 1.4.',
        interlaced: 'De voorkeurstiming is geïnterlinieerd.',
        tooLarge:
          'De voorkeurstiming is groter dan 1920x1080 bij 60 Hz, meer dan NanoKVM kan vastleggen.',
        recovery: 'Herstel',
        recoveryNeeded:
          'De laatste schrijfactie is niet geverifieerd, dus het EDID-gebied van de videochip verkeert in een onbekende staat. Herstel de fabrieks-EDID om weer een bekende staat te krijgen.',
        recoveryDesc:
          'Schrijf een bekende EDID terug naar de videochip als een toegepaste EDID de host zonder beeld heeft achtergelaten.',
        restoreFactory: 'Fabrieks-EDID herstellen',
        restoreBackup: 'Vorige EDID herstellen',
        restoreTitle: 'Deze EDID herstellen?',
        restoreFactoryTarget: 'De fabrieks-EDID waarmee NanoKVM is geleverd.',
        restoreBackupTarget: 'De meest recente back-up, toegepast op {{time}}.',
        restoreNotice:
          'Een herstel wordt op dezelfde manier geschreven als een toepassing en heeft dezelfde gevolgen.',
        restored: 'EDID hersteld en geverifieerd.',
        restoreFailed: 'Het herstellen van de EDID is mislukt.',
        mismatchTitle: 'Geschreven en teruggelezen',
        mismatchWritten: 'Geschreven',
        mismatchRead: 'Teruggelezen',
        restoreOkBtn: 'Herstellen',
        hardware: 'Gedetecteerde hardware: {{hardware}}',
        hardwareUnknown: 'Onbekend',
        confirmWord: 'TOEPASSEN',
        confirmPrompt: 'Typ {{word}} om de knop Toepassen in te schakelen.',
        okBtn: 'Toepassen',
        cancelBtn: 'Annuleren'
      },
      presentation: {
        title: 'USB-presentatie',
        loading: 'Laden...',
        current: 'Huidige USB-presentatie',
        noProfile: 'Geen profiel toegepast',
        linked: 'Gekoppelde functies',
        hostState: 'USB van de host',
        hostUnbound: 'Controller niet gebonden',
        hdmiState: 'HDMI-ingang',
        hdmiSignal: 'Signaal aanwezig',
        hdmiUnreported: 'Nog geen melding van de opname',
        endpoints: 'Endpoints',
        fifos: 'FIFO-plekken',
        pending: 'Openstaande wijzigingen',
        pendingEdits: 'Niet-opgeslagen identiteitswijzigingen',
        pendingProfile: '{{profile}} is geselecteerd maar niet toegepast',
        pendingNone: 'Geen',
        lastApply: 'Laatste toepassing',
        applyFailed: 'Mislukt op {{profile}} om {{time}}',
        applyClean: 'Geen fout geregistreerd',
        lastKnownGood: 'Laatst bekende werkende stand',
        rollbackTarget: 'Doel van terugdraaien',
        rollbackNone: 'Geen',
        powerCyclePending:
          'De controller is aan de host onttrokken. Zet de aangesloten computer uit en weer aan om het apparaat terug te krijgen.',
        rollback: 'Terugdraaien',
        rollbackTitle: 'Terugdraaien naar {{profile}}?',
        rollbackDesc: 'De gadget wordt opnieuw geënumereerd; USB-functies vallen even weg.',
        profile: 'USB-profiel',
        builtIn: 'ingebouwd',
        descriptors: 'descriptors',
        imported: 'geïmporteerd',
        clone: 'Klonen',
        cloneTitle: 'Dit profiel klonen',
        cloneToEdit:
          'Ingebouwde profielen blijven alleen-lezen. Kloon dit profiel om de identiteit te bewerken.',
        profileName: 'Profielnaam',
        profileNameHint: 'Kleine letters, cijfers, punten, liggende streepjes en koppeltekens.',
        import: 'Pakket importeren',
        export: 'Pakket exporteren',
        delete: 'Verwijderen',
        deleteTitle: 'Dit profiel verwijderen?',
        deleteDesc: '{{profile}} van de NanoKVM verwijderen.',
        identity: 'USB-identiteit',
        preset: 'Vooraf ingestelde identiteit',
        presetPlaceholder: 'Identiteit van een bekend apparaat overnemen',
        presetHint:
          'Een voorinstelling vult de Vendor ID, de Product ID en de twee naamvelden. Descriptors brengt ze niet mee.',
        presetSource: 'Identiteit zoals vastgelegd in {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Deze Vendor ID hoort bij een andere fabrikant',
        productId: 'Product ID',
        bcdUSB: 'USB-versie',
        bcdDevice: 'Apparaatversie',
        manufacturer: 'Fabrikant',
        product: 'Product',
        serial: 'Serienummer',
        configuration: 'Configuratiereeks',
        hidLayout: 'HID-apparaten',
        hidRoleKeyboard: 'Toetsenbord',
        hidRoleRelative: 'Muis (relatief)',
        hidRoleAbsolute: 'Aanwijzer (absoluut)',
        hidOff: 'Niet aanwezig',
        hidInterface: 'Interface {{index}}',
        hidBootKeyboardShared:
          'Het toetsenbord deelt een interface en biedt daarom geen rapport in bootprotocol meer. Sommige BIOS- en UEFI-installaties zien het dan niet.',
        functions: 'Functies',
        descriptorAssets: 'Opgeslagen descriptors: {{count}}',
        endpointUse:
          'IN {{inUse}} in gebruik, {{inFree}} vrij; OUT {{outUse}} in gebruik, {{outFree}} vrij',
        apply: 'Toepassen',
        applyTitle: 'Dit USB-profiel toepassen?',
        applyDesc: 'De NanoKVM presenteert {{profile}} aan de aangesloten computer.',
        reconnect:
          'Toetsenbord, muis en andere USB-functies vallen even weg terwijl de gadget opnieuw wordt gekoppeld.',
        applyLinks: 'Koppelt: {{functions}}',
        applyRemoves: 'Verwijdert: {{functions}}',
        applyNoHid:
          'Na deze toepassing blijft er geen HID-functie over. Toetsenbord en muis werken dan niet meer.',
        applyRollback: 'Een mislukte toepassing keert terug naar {{profile}}.',
        recoveryPowerCycle:
          'Geen enkele HID overleeft deze toepassing, dus een host die niet meer reageert is alleen te herstellen door hem uit en weer aan te zetten.',
        recoveryReboot:
          'Er verdwijnt een interface uit het samengestelde apparaat; de host heeft mogelijk een herstart nodig om de rest opnieuw te binden.',
        recoveryHdmiReset:
          'Een videofunctie wordt opnieuw opgebouwd, waardoor de opnameketen erachter reset.',
        recoveryReconnect: 'De host enumereert het apparaat opnieuw; USB-functies vallen even weg.',
        cancel: 'Annuleren',
        noFunctions: 'Geen gekoppelde functies',
        loadFailed: 'De presentatieprofielen konden niet worden geladen',
        operationFailed: 'De presentatiebewerking is mislukt'
      },
      passthrough: {
        title: 'USB-passthrough',
        loading: 'Laden...',
        mode: 'Modus',
        hybrid: 'Hybride',
        exact: 'Exact',
        hybridDesc: 'Behoudt het boottoetsenbord en de relatieve muis, voor compatibele apparaten.',
        exactDesc: 'Vervangt elke USB-functie van NanoKVM door het geïmporteerde apparaat.',
        hybridWarning: 'Hybride houdt het toetsenbord en de relatieve muis beschikbaar',
        hybridWarningDesc:
          'Opslag, USB-netwerk en de absolute aanwijzer vallen weg zolang de geïmporteerde functie actief is.',
        hidWarning: 'Passthrough starten geeft het toetsenbord, de muis en virtuele media op',
        hidWarningDesc:
          'NanoKVM heeft maar één USB-apparaatcontroller en de proxy heeft die helemaal nodig. Zolang een sessie loopt ziet de externe host daarom het doorgegeven apparaat in plaats van het toetsenbord, de muis en de virtuele media van NanoKVM. Ze komen vanzelf terug zodra de sessie stopt. Deze webinterface merkt er niets van, dus u kunt een sessie altijd vanaf deze pagina stoppen.',
        hidWarningSafeDesc:
          'NanoKVM heeft maar één USB-apparaatcontroller en de proxy heeft die helemaal nodig. Zolang een sessie loopt ziet de externe host daarom het doorgegeven apparaat in plaats van het toetsenbord, de muis en de virtuele media van NanoKVM. Ze komen terug zodra de sessie stopt.',
        isoLabel: 'Isochrone overdrachten toestaan',
        isoHint:
          'Laat webcams, microfoons en andere streamende apparaten door. Niemand heeft gemeten wat deze hardware aankan.',
        isoWarning:
          'Isochroon streamen is hier onbeproefd en kan het toetsenbord en de muis vasthouden tot je de sessie stopt',
        info: {
          title: 'Info',
          hybrid:
            'De hybride modus houdt het toetsenbord en de relatieve muis beschikbaar. Opslag, USB-netwerk en de absolute aanwijzer vallen weg zolang het geïmporteerde apparaat actief is.',
          exact:
            'De exacte modus vervangt elke USB-functie van NanoKVM door het geïmporteerde apparaat. Het toetsenbord, de muis en de virtuele media komen vanzelf terug zodra de sessie stopt.',
          udc: 'NanoKVM heeft maar één USB-apparaatcontroller en de proxy heeft die helemaal nodig; daarom verdwijnen de functies hierboven zolang een sessie loopt.',
          web: 'Deze webinterface merkt er niets van, dus u kunt een sessie altijd vanaf deze pagina stoppen.',
          network:
            'Start passthrough via ethernet of wifi. Starten vanaf het USB-netwerk van NanoKVM wordt geweigerd, omdat die verbinding zou verdwijnen.',
          iso: 'Webcams, microfoons en andere isochrone apparaten worden geweigerd zolang u isochrone overdrachten niet toestaat. Die weg werkt, maar is nooit gemeten op deze hardware: beschouw de doorvoer als onbekend.',
          camera:
            'De camera en microfoon van de browser, onder Apparaten, blijven de beproefde manier om de host er een te geven.'
        },
        session: 'Sessie',
        activeDesc: 'Er is een apparaat geïmporteerd en de proxy houdt de USB-controller vast.',
        inactiveDesc:
          'Er loopt geen sessie. Het toetsenbord, de muis en virtuele media werken normaal.',
        device: 'Apparaat',
        busId: 'Bus-ID',
        speed: 'Snelheid',
        exporter: 'Exporter',
        local: 'Geïmporteerd als',
        localValue: 'Bus {{bus}}, adres {{address}}',
        udc: 'USB-controller',
        pid: 'Proxy-PID',
        startedAt: 'Gestart',
        isoDevice:
          'Dit apparaat streamt over isochrone eindpunten, wat op deze hardware nooit is gemeten',
        exporterLabel: 'Adres van de exporter',
        exporterHint:
          'De host en poort die NanoKVM belt. Via de tunnel hieronder is dat {{exporter}}.',
        busIdLabel: 'Bus-ID op uw eigen machine',
        busIdHint: 'De busid die usbip list -l voor het apparaat toont, bijvoorbeeld {{example}}.',
        start: 'Passthrough starten',
        stop: 'Passthrough stoppen',
        startTitle: 'USB-passthrough starten?',
        startDevice: 'NanoKVM importeert {{busId}} van {{exporter}}.',
        startHid:
          'Het USB-toetsenbord, de muis en virtuele media werken niet zolang de sessie loopt en werken vanzelf weer zodra u die stopt.',
        startIso:
          'Webcams en andere isochrone apparaten vereisen dat je de isochrone schakelaar aanzet voordat je start.',
        startWeb:
          'Deze webinterface blijft werken, dus u kunt de sessie op elk moment vanaf deze pagina stoppen.',
        startNetwork:
          'Gebruik deze pagina via ethernet of wifi. Starten vanaf het USB-netwerk van NanoKVM wordt geweigerd omdat die verbinding zou verdwijnen.',
        okBtn: 'Starten',
        cancelBtn: 'Annuleren',
        instructions: 'Op uw eigen machine',
        instructionsDesc:
          'Er is bewust geen clientagent om te installeren. Voer deze gewone usbip-commando’s uit op de machine waar het apparaat aan hangt.',
        copyFailed: 'Kopiëren mislukt. Kopieer het commando handmatig.',
        copyInsecure:
          'Deze pagina wordt niet via HTTPS geleverd, daarom blokkeert de browser het kopiëren. Kopieer de opdracht handmatig of schakel HTTPS in bij Instellingen, Netwerk.',
        directNote:
          'Zonder tunnel moet usbipd bereikbaar zijn op uw netwerk en moet het exporteradres hierboven daarnaar verwijzen. usbip draagt het apparaat onversleuteld over, dus de tunnel verdient de voorkeur.',
        steps: {
          modprobe: {
            title: 'Laad het stuurprogramma van de exportkant',
            desc: 'usbip-host laat uw kernel een lokaal apparaat afstaan. Het wordt niet standaard geladen.'
          },
          list: {
            title: 'Zoek het apparaat',
            desc: 'Toont elk lokaal apparaat met zijn busid en zijn fabrikant:product-paar. Noteer de busid van het gewenste apparaat.'
          },
          bind: {
            title: 'Koppel het aan usbip',
            desc: 'Neemt het apparaat weg bij het gewone stuurprogramma, dus het werkt op deze machine niet meer tot u het ontkoppelt.'
          },
          serve: {
            title: 'Bied het aan',
            desc: 'usbipd blijft op de voorgrond draaien en wacht tot NanoKVM het apparaat importeert.',
            notice:
              'De gewone usbipd heeft geen optie voor een luisteradres en luistert op alle interfaces. Houd poort {{port}} dicht in uw firewall en laat alleen de tunnel hieronder erbij.'
          },
          tunnel: {
            title: 'Richt het op NanoKVM',
            desc: 'Een omgekeerde SSH-tunnel: poort {{port}} op de loopback van NanoKVM zelf wordt de usbipd op deze machine. Laat hem de hele sessie draaien.'
          },
          exporter: {
            title: 'Gebruik dit als exporter',
            desc: 'Zet dit in het exporterveld hierboven, vul de bus-ID in en start de sessie.'
          },
          unbind: {
            title: 'Geef het apparaat terug',
            desc: 'Nadat de sessie is gestopt, geeft dit het apparaat terug aan het gewone stuurprogramma op deze machine.'
          }
        }
      },
      mcp: {
        title: 'MCP-service',
        service: 'MCP-afstandsbediening',
        serviceDesc:
          'Vertrouwde MCP-clients toestaan het toetsenbord en de muis te bedienen en schermafbeeldingen te maken',
        securityWarning:
          'Iedereen met deze API-sleutel kan de externe host bedienen en het scherm bekijken. Gebruik HTTPS en schakel de service alleen in op vertrouwde netwerken.',
        endpoint: 'Eindpunt',
        apiKey: 'API-sleutel',
        regenerateConfirmTitle: 'MCP API-sleutel opnieuw genereren?',
        regenerateConfirmDesc: 'De huidige sleutel werkt dan onmiddellijk niet meer.',
        enableConfirmTitle: 'Externe MCP-bediening inschakelen?',
        enableConfirmDesc:
          'Als MCP wordt ingeschakeld, stopt PicoClaw en worden alle actieve PicoClaw-sessies gesloten.',
        failed: 'MCP-bewerking mislukt',
        copyFailed: 'Kopiëren mislukt. Kopieer handmatig.',
        copyInsecure:
          'Deze pagina wordt niet via HTTPS geleverd, daarom blokkeert de browser het kopiëren. Kopieer handmatig of schakel HTTPS in bij Instellingen, Netwerk.',
        okBtn: 'Bevestigen',
        cancelBtn: 'Annuleren'
      },
      about: {
        title: 'Over NanoKVM',
        information: 'Informatie',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applicatie versie',
        applicationTip: 'Versie van de NanoKVM-webapplicatie',
        image: 'Image versie',
        imageTip: 'Versie van de NanoKVM-systeemimage',
        deviceKey: 'Apparaat sleutel',
        community: 'Community',
        hostname: 'Hostnaam',
        hostnameUpdated: 'Hostnaam bijgewerkt. Start opnieuw op om toe te passen.',
        ipType: {
          Wired: 'Bedraad',
          Wireless: 'Draadloos',
          Other: 'Anders'
        }
      },
      appearance: {
        title: 'Uiterlijk',
        display: 'Beeldscherm',
        language: 'Taal',
        languageDesc: 'Selecteer de taal voor de interface',
        webTitle: 'Webtitel',
        webTitleDesc: 'Pas de titel van de webpagina aan',
        favicon: 'Favicon',
        faviconDesc: 'Pas het pictogram van het browsertabblad aan',
        faviconPlaceholder: 'Afbeeldings-URL',
        faviconUpload: 'Uploaden',
        faviconReset: 'Herstellen',
        faviconCustom: 'Aangepast pictogram',
        faviconBoot: 'Pictogram uit /boot/logo.ico',
        faviconDefault: 'Standaardpictogram',
        faviconOverridesBoot: 'Overschrijft /boot/logo.ico',
        faviconErrUrl: 'Voer een http:// of https:// afbeeldingsadres in',
        faviconErrFetch: 'Het apparaat kon die afbeelding niet downloaden',
        faviconErrLarge: 'Afbeelding is te groot. De limiet is 256 KB',
        faviconErrType: 'Niet-ondersteunde afbeelding. Gebruik .ico, .png, .svg, .gif of .jpg',
        faviconErrSave: 'Kan het pictogram niet opslaan',
        menuBar: {
          title: 'Menubalk',
          mode: 'Weergavemodus',
          modeDesc: 'Geef de menubalk weer op het scherm',
          modeOff: 'Uit',
          modeAuto: 'Automatisch verbergen',
          modeAlways: 'Altijd zichtbaar',
          keyboardLedStatus: 'Toetsvergrendelingsindicatoren',
          keyboardLedStatusDesc:
            'Toon de Num Lock-, Caps Lock- en Scroll Lock-status van de externe computer',
          icons: 'Submenupictogrammen',
          iconsDesc: 'Submenupictogrammen weergeven in de menubalk'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Toetsvergrendelingsstatus van extern toetsenbord',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Aan',
        off: 'Uit',
        unknown: 'Onbekend'
      },
      device: {
        title: 'Apparaat',
        oled: {
          title: 'OLED',
          description: 'OLED scherm automatisch slapen',
          0: 'Nooit',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 uur'
        },
        ssh: {
          description: 'Schakel SSH externe toegang in',
          tip: 'Stel een sterk wachtwoord in voordat u (Account - Wachtwoord wijzigen) inschakelt'
        },
        advanced: 'Geavanceerde instellingen',
        swap: {
          title: 'Wisselen',
          disable: 'Uitschakelen',
          description: 'Stel de grootte van het wisselbestand in',
          tip: 'Het inschakelen van deze functie kan de bruikbare levensduur van uw SD-kaart verkorten!'
        },
        mouseJiggler: {
          title: 'Muisschommel',
          description: 'Voorkom dat de externe host in slaap valt',
          disable: 'Uitschakelen',
          absolute: 'Absolute modus',
          relative: 'Relatieve modus'
        },
        mdns: {
          description: 'Schakel de mDNS detectieservice in',
          tip: 'Schakel het uit als het niet nodig is'
        },
        hdmi: {
          description: 'Schakel HDMI/monitoruitgang in',
          idleTimeoutTitle: 'Time-out voor inactieve opname',
          idleTimeoutDescription:
            'HDMI-opname stoppen nadat er gedurende deze tijd geen actieve kijkers zijn:',
          minutes: 'min'
        },
        autostart: {
          title: 'Instellingen voor automatisch starten van scripts',
          description:
            'Beheer scripts die automatisch worden uitgevoerd bij het opstarten van het systeem',
          new: 'Nieuw',
          deleteConfirm: 'Weet u zeker dat u dit bestand wilt verwijderen?',
          yes: 'Ja',
          no: 'Nee',
          scriptName: 'Scriptnaam automatisch starten',
          scriptContent: 'Scriptinhoud automatisch starten',
          settings: 'Instellingen'
        },
        hidOnly: 'HID-Alleen modus',
        hidOnlyDesc:
          'Stop met het emuleren van virtuele apparaten en behoud alleen de basisbesturing van HID',
        disk: 'Virtuele schijf',
        diskDesc: 'Koppel virtuele U-schijf aan de externe host',
        rebindNotice:
          'Deze schakelaar omzetten laat het USB-apparaat opnieuw opsommen, waardoor het doel kort zijn virtuele apparaten en zijn USB-netwerk kwijt is.',
        media: {
          title: 'Camera- en microfoonplaatsen',
          desc: 'Geef aan welke media-apparaten browsers mogen invullen. Het endpointbudget wordt gecontroleerd zodra het USB-profiel wordt toegepast. Bij opslaan wordt het apparaat opnieuw opgesomd en worden verbonden browsers verbroken.',
          cameras: "Camera's",
          microphones: 'Microfoons',
          name: 'Naam',
          namePlaceholder: 'Wordt op de doelcomputer getoond',
          addCamera: 'Camera toevoegen',
          addMicrophone: 'Microfoon toevoegen',
          remove: 'Verwijderen',
          cameraDefault: 'NanoKVM-camera {{index}}',
          microphoneDefault: 'NanoKVM-microfoon {{index}}',
          nameRequired: 'Elke plaats heeft een naam nodig.',
          budgetHint:
            'De zes USB-IN-endpoints zijn een vaste hardwaregrens. Zet toetsenbord, muis en absolute aanwijzer op één HID-interface onder USB-presentatie, of schakel hier de virtuele schijf uit of onder Netwerk de USB-netwerkadapter.',
          unsupported:
            'Deze kernel kan media-apparaten geen naam geven, dus computers tonen de standaardnaam.',
          save: 'Plaatsen opslaan',
          disconnect: 'Verbreken',
          disconnectAll: 'Alle bronnen verbreken',
          limit: 'Camera- en microfoonplaatsen mogen samen niet meer dan acht zijn.',
          failed: 'De mediaplaatsen konden niet worden bijgewerkt.'
        },
        reboot: 'Opnieuw opstarten',
        rebootDesc: 'Weet u zeker dat u NanoKVM opnieuw wilt opstarten?',
        okBtn: 'Ja',
        cancelBtn: 'Nee'
      },
      network: {
        title: 'Netwerk',
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi configureren',
          apMode: 'AP-modus is ingeschakeld, maak verbinding met Wi-Fi door de QR-code te scannen',
          connect: 'Wi-Fi verbinden',
          connectDesc1: 'Voer de netwerk-SSID en het wachtwoord in',
          connectDesc2: 'Voer het wachtwoord in om met dit netwerk te verbinden',
          disconnect: 'Weet je zeker dat je de netwerkverbinding wilt verbreken?',
          failed: 'Verbinding mislukt, probeer het opnieuw.',
          ssid: 'Naam',
          password: 'Wachtwoord',
          joinBtn: 'Verbinden',
          confirmBtn: 'OK',
          cancelBtn: 'Annuleren'
        },
        tls: {
          description: 'HTTPS-protocol inschakelen',
          tip: 'Let op: HTTPS gebruiken kan de latentie verhogen, vooral in MJPEG-videomodus.'
        },
        usb: {
          title: 'USB-netwerkadapter',
          desc: 'Geeft de bestuurde computer een netwerkkaart via USB',
          type: 'Adaptertype',
          typeDesc: 'NCM voor moderne systemen, RNDIS voor oudere Windows'
        },
        bridge: {
          title: 'De adapter is verbonden met',
          lan: 'Jouw netwerk',
          kvmOnly: 'Alleen NanoKVM',
          lanDesc:
            'De computer komt via de ethernetpoort van de NanoKVM op jouw netwerk en krijgt een eigen adres van je router.',
          kvmOnlyDesc:
            'De computer krijgt zijn adres van de NanoKVM en bereikt de NanoKVM, maar niets daarbuiten.',
          loading: 'Laden...',
          state: 'Status',
          states: {
            disabled: 'Alleen NanoKVM',
            enabled: 'Jouw netwerk',
            rolledBack: 'Teruggedraaid',
            failed: 'Mislukt',
            pending: 'Bezig'
          },
          uplink: 'Uplink',
          ports: 'Poorten',
          up: 'actief',
          down: 'inactief',
          noLink: 'geen link',
          enableTitle: 'De computer met jouw netwerk verbinden?',
          disableTitle: 'De computer tot alleen de NanoKVM beperken?',
          reconnect:
            'De beheerverbinding wordt kort verbroken en hersteld terwijl het adres verhuist.',
          rollback:
            'Als de verificatie mislukt, wordt de vorige configuratie automatisch hersteld.',
          enableBtn: 'Verbind met mijn netwerk',
          disableBtn: 'Alleen NanoKVM',
          cancelBtn: 'Annuleren',
          interrupted:
            'De verbinding werd tijdens het toepassen verbroken. De huidige status wordt opnieuw gecontroleerd.',
          pendingNotice: 'Een wijziging van de brug is nog bezig of is voortijdig afgebroken.',
          revert: 'Vorige configuratie herstellen',
          rolledBackNotice:
            'De laatste wijziging is teruggedraaid en de vorige configuratie is hersteld.',
          verifyFailed: 'Verificatie mislukt: {{gates}}',
          gates: {
            address: 'adres',
            gateway: 'gateway',
            inbound: 'inkomende verbinding'
          },
          inboundWeak:
            'De inkomende controle slaagde alleen doordat NanoKVM verbinding met zichzelf maakte. Dat bewijst dat de webdienst luistert en lokaal bereikbaar is, niet dat een verzoek vanaf het netwerk aankomt.',
          noCarrier:
            'Geen link op {{port}}. De brug heeft geen pad naar het netwerk zolang er geen kabel is aangesloten.',
          loop: 'De router wordt ook op {{port}} geleerd, dus die poort is een tweede pad naar hetzelfde netwerk. Spanning tree staat uit, dus niets hier verbreekt de lus: koppel een van de twee paden los.',
          failedNotice:
            'De laatste wijziging kon niet ongedaan worden gemaakt. NanoKVM is mogelijk alleen bereikbaar via het wifi-toegangspunt of een seriële console.'
        },
        dns: {
          title: 'DNS',
          description: 'Configureer DNS-servers voor NanoKVM',
          mode: 'Modus',
          dhcp: 'DHCP',
          manual: 'Handmatig',
          add: 'DNS toevoegen',
          save: 'Opslaan',
          invalid: 'Voer een geldig IP-adres in',
          noDhcp: 'Er is momenteel geen DHCP-DNS beschikbaar',
          saved: 'DNS-instellingen opgeslagen',
          saveFailed: 'DNS-instellingen opslaan mislukt',
          unsaved: 'Niet-opgeslagen wijzigingen',
          maxServers: 'Maximaal {{count}} DNS-servers toegestaan',
          dnsServers: 'DNS-servers',
          dhcpServersDescription: 'DNS-servers worden automatisch via DHCP verkregen',
          manualServersDescription: 'DNS-servers kunnen handmatig worden bewerkt',
          networkDetails: 'Netwerkdetails',
          interface: 'Interface',
          ipAddress: 'IP-adres',
          subnetMask: 'Subnetmasker',
          router: 'Router',
          none: 'Geen'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC-server',
        description:
          'Laat elke VNC-client het externe scherm bekijken en het toetsenbord en de muis gebruiken, met uw NanoKVM-account als aanmelding',
        port: 'Poort',
        portDescription: 'Maak verbinding met deze poort op het NanoKVM-adres'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Geheugen optimalisatie',
          tip: 'Wanneer geheugen gebruik de limiet overschreid, garbage collection wordt agressiever uitgevoerd om geheugen vrij te maken. geadviseerd om 50MB te kiezen als Tailscale wordt gebruikt. Tailscale moet worden herstart om de wijziging door te voeren.'
        },
        swap: {
          title: 'Geheugen wisselen',
          tip: 'Als de problemen aanhouden nadat u geheugenoptimalisatie hebt ingeschakeld, probeer dan het wisselgeheugen in te schakelen. Hierdoor wordt de grootte van het wisselbestand standaard ingesteld op 256MB, wat kan worden aangepast in "Instellingen > Apparaat".'
        },
        restart: 'Weet u zeker dat u Tailscale opnieuw wilt opstarten?',
        stop: 'Weet u zeker dat u Tailscale wilt stoppen?',
        stopDesc: 'Meld Tailscale af en schakel het automatisch opstarten bij het opstarten uit.',
        loading: 'Laden...',
        notInstall: 'Tailscale niet gevonden! Installeer a.u.b.',
        install: 'Installeren',
        installing: 'Installeren bezig',
        failed: 'Installatie mislukt',
        retry: 'Vernieuw en probeer opnieuw. Of probeer handmatig te installeren',
        download: 'Download het',
        package: 'installatiepakket',
        unzip: 'en pak het uit',
        upTailscale: 'Upload tailscale naar NanoKVM directory /usr/bin/',
        upTailscaled: 'Upload tailscaled naar NanoKVM directory /usr/sbin/',
        refresh: 'Vernieuw huidige pagina',
        notRunning: 'Tailscale is niet actief. Start het programma om door te gaan.',
        run: 'Begin',
        notLogin:
          'Het apparaat is nog niet gekoppeld. Log in en koppel dit apparaat aan uw account.',
        urlPeriod: 'Deze url is 10 minuten geldig',
        login: 'Inloggen',
        loginSuccess: 'Inloggen gelukt',
        enable: 'Tailscale inschakelen',
        deviceName: 'Apparaatnaam',
        deviceIP: 'Apparaat IP',
        account: 'Account',
        logout: 'Uitloggen',
        logoutDesc: 'Weet u zeker dat u wilt uitloggen?',
        uninstall: 'Verwijderen Tailscale',
        uninstallDesc: 'Weet u zeker dat u Tailscale wilt verwijderen?',
        okBtn: 'Ja',
        cancelBtn: 'Nee'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Laden...',
        notInstall: 'Niet geïnstalleerd',
        notConfigured: 'Niet geconfigureerd',
        stopped: 'Gestopt',
        running: 'Actief',
        connected: 'Verbonden',
        error: 'Fout',
        atBoot: 'start bij opstarten',
        notAtBoot: 'start niet bij opstarten',
        arguments: 'Argumenten',
        argumentsTip:
          'Opdrachtregelargumenten die bij het starten aan de service worden meegegeven.',
        env: 'Omgevingsvariabelen',
        envKey: 'Naam',
        envValue: 'Waarde',
        envAdd: 'Variabele toevoegen',
        envRemove: 'Verwijderen',
        configured: 'Ingesteld',
        save: 'Opslaan',
        saved: 'Configuratie opgeslagen',
        start: 'Starten',
        stop: 'Stoppen',
        restart: 'Opnieuw starten',
        logs: 'Logboek',
        logsEmpty: 'Nog geen logregels',
        refresh: 'Vernieuwen',
        binary: 'Binair bestand',
        binaryShipped: 'Meegeleverd met de firmware',
        binaryCustom: 'Eigen binair bestand',
        binaryUpload: 'Binair bestand uploaden',
        binaryRevert: 'Meegeleverd bestand herstellen',
        binaryRevertDesc:
          'Het geüploade binaire bestand verwijderen en de meegeleverde versie herstellen?',
        serverWarning: 'Een server zonder beperkingen werkt als open proxy',
        noHealthSignal:
          'Deze dienst meldt geen gezondheidsstatus, dus NanoKVM weet alleen dat het proces draait, niet of de tunnel verbonden is.',
        memoryWarning: 'Meerdere toegangsdiensten tegelijk uitvoeren kan het geheugen uitputten',
        resources: 'Bronnen',
        memory: {
          title: 'Geheugenlimiet',
          description:
            'Beperkt de Go-heap van newt vanaf de volgende herstart tot {{limit}} MiB. Zijn eigen limiet, niet die van Tailscale; uit betekent de standaard van Go, met GOGC=50 in beide gevallen.',
          noRuntime:
            'wstunnel is Rust: geen garbagecollector en geen heaplimiet om in te stellen, en zijn werkthreads volgen al de enige CPU van het apparaat.',
          notApplicable: 'Niet van toepassing'
        },
        swap: {
          title: 'Wisselbestand',
          description:
            'Zet een wisselbestand van 256 MB op de SD-kaart. Systeembreed: dezelfde swap bedient Tailscale, de KVM-server en al het andere op het apparaat.'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nee'
      },
      update: {
        title: 'Controleren op updates',
        queryFailed: 'Ophalen versie mislukt',
        updateFailed: 'Update mislukt. Probeer het opnieuw.',
        isLatest: 'U heeft al de nieuwste versie.',
        rebooting:
          'De nieuwe kernel wordt geïnstalleerd en het apparaat start opnieuw op. Dit kan enkele minuten duren; schakel de voeding niet uit.',
        kernelUpdate:
          'Deze update installeert kernel {{version}}. Het apparaat start opnieuw op en keert vanzelf terug naar de huidige kernel als de nieuwe niet opstart.',
        rolledBack:
          'Kernel {{version}} is niet opgestart en het apparaat is teruggevallen op de vorige kernel.',
        available: 'Er is een update beschikbaar. Weet u zeker dat u wilt updaten?',
        updating: 'Update gestart. Even geduld a.u.b...',
        confirm: 'Bevestigen',
        cancel: 'Annuleren',
        preview: 'Preview-updates',
        previewDesc: 'Krijg vroegtijdig toegang tot nieuwe functies en verbeteringen',
        previewTip:
          'Houd er rekening mee dat preview-releases bugs of onvolledige functionaliteit kunnen bevatten!',
        customServer: {
          title: 'Aangepaste updateserver',
          desc: 'Online-updates zoeken en downloaden vanaf een opgegeven server',
          invalidUrl:
            'Voer een geldige HTTP- of HTTPS-servermap in zonder queryparameters, fragment of latest.json.',
          loadFailed: 'De configuratie van de updateserver kon niet worden geladen.',
          saveFailed: 'De configuratie van de updateserver kon niet worden opgeslagen.',
          saved: 'De configuratie van de updateserver is opgeslagen.',
          save: 'Opslaan',
          confirmTitle: 'Een aangepaste updateserver gebruiken?',
          confirmDesc:
            'SHA-512 controleert alleen of het pakket overeenkomt met het manifest dat door deze server wordt verstrekt. Het bewijst niet dat het pakket een officiële NanoKVM-release is. Een defecte of kwaadwillende server kan het apparaat onbruikbaar maken, gegevensverlies veroorzaken of het systeem compromitteren.',
          confirm: 'Toch gebruiken',
          previewDisabled:
            'Preview-updates zijn niet beschikbaar zolang een aangepaste updateserver is ingeschakeld.'
        },
        offline: {
          kernelNotice:
            'Dit pakket bevat een kernel. Die wordt naar het reserveslot geschreven en het apparaat start opnieuw op om hem te proberen; komt hij niet terug, dan keert het apparaat vanzelf terug naar de huidige kernel.',
          kernelConfirm: 'Kernel installeren',
          kernelCancel: 'Annuleren',
          title: 'Offline-updates',
          desc: 'Update via lokaal installatiepakket',
          upload: 'Uploaden',
          checksumPlaceholder: 'SHA-256-controlesom (optioneel)',
          invalidChecksum: 'De SHA-256-controlesom moet 64 hexadecimale tekens bevatten.',
          checksumMismatch: 'De SHA-256-verificatie is mislukt. Het pakket is mogelijk beschadigd.',
          invalidName: 'Ongeldig bestandsnaamformaat. Download de versie van GitHub-releases.',
          updateFailed: 'Update mislukt. Probeer het opnieuw.'
        }
      },
      account: {
        title: 'Account',
        webAccount: 'Web Account Naam',
        role: 'Rol',
        roles: {
          admin: 'Beheerder',
          user: 'Gebruiker'
        },
        password: 'Wachtwoord',
        updateBtn: 'Update',
        logoutBtn: 'Afmelden',
        logoutDesc: 'Weet u zeker dat u wilt uitloggen?',
        okBtn: 'Ja',
        cancelBtn: 'Nee',
        users: {
          title: 'Gebruikers',
          create: 'Gebruiker aanmaken',
          enabled: 'Ingeschakeld',
          disabled: 'Uitgeschakeld',
          deviceOwner: 'Eigenaar van het apparaat',
          resetPassword: 'Wachtwoord opnieuw instellen',
          delete: 'Verwijderen',
          deleteConfirm: 'Deze gebruiker verwijderen en al zijn sessies intrekken?',
          created: 'Gebruiker aangemaakt',
          deleted: 'Gebruiker verwijderd',
          passwordUpdated: 'Wachtwoord bijgewerkt',
          loadFailed: 'Gebruikers laden mislukt',
          saveFailed: 'Gebruiker opslaan mislukt',
          deleteFailed: 'Gebruiker verwijderen mislukt'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistent',
      empty: 'Open het paneel en start een taak om te beginnen.',
      inputPlaceholder: 'Beschrijf wat u wilt dat de PicoClaw doet',
      newConversation: 'Nieuw gesprek',
      processing: 'Verwerken...',
      agent: {
        defaultTitle: 'Algemene assistent',
        defaultDescription: 'Algemene hulp bij chatten, zoeken en werkruimte.',
        kvmTitle: 'Bediening op afstand',
        kvmDescription: 'Bedien de externe host via NanoKVM.',
        switched: 'Agentrol gewijzigd',
        switchFailed: 'Kan agentrol niet wisselen'
      },
      send: 'Verzenden',
      cancel: 'Annuleren',
      status: {
        connecting: 'Verbinden met gateway...',
        connected: 'PicoClaw-sessie verbonden',
        disconnected: 'PicoClaw-sessie gesloten',
        stopped: 'Stopverzoek verzonden',
        runtimeStarted: 'PicoClaw runtime gestart',
        runtimeStartFailed: 'Kan PicoClaw runtime niet starten',
        runtimeStopped: 'PicoClaw runtime gestopt',
        runtimeStopFailed: 'Kan PicoClaw runtime niet stoppen',
        controlSwitchedToMCP: 'Bediening overgeschakeld naar de externe MCP-service'
      },
      connection: {
        runtime: {
          checking: 'Controleren',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime gereed',
          stopped: 'Runtime gestopt',
          blockedByMCP: 'Externe MCP-bediening is actief',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime niet beschikbaar',
          configError: 'Configuratiefout'
        },
        transport: {
          connecting: 'Verbinden',
          connected: 'Verbonden',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Inactief',
          busy: 'Bezet'
        }
      },
      message: {
        toolAction: 'Actie',
        observation: 'Observatie',
        screenshot: 'Schermafbeelding'
      },
      overlay: {
        locked: 'PicoClaw bestuurt het apparaat. Handmatige invoer is gepauzeerd.'
      },
      control: {
        picoclaw: 'Apparaatbediening: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Apparaatbediening: externe MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Apparaatbediening: uit',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Bediening geven',
        release: 'Vrijgeven',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw-bediening gegeven',
        released: 'PicoClaw-bediening vrijgegeven',
        grantFailed: 'Kan PicoClaw-bediening niet geven',
        releaseFailed: 'Kan PicoClaw-bediening niet vrijgeven',
        grantConfirmTitle: 'Apparaatbediening overschakelen naar PicoClaw?',
        grantConfirmDesc: 'Schrijfacties van de externe MCP naar het apparaat worden onderbroken.'
      },
      install: {
        install: 'PicoClaw installeren',
        installing: 'PicoClaw installeren',
        success: 'PicoClaw is succesvol geïnstalleerd',
        failed: 'Kan PicoClaw niet installeren',
        uninstalling: 'Runtime verwijderen...',
        uninstalled: 'Runtime is succesvol verwijderd.',
        uninstallFailed: 'Verwijderen mislukt.',
        requiredTitle: 'PicoClaw is niet geïnstalleerd',
        requiredDescription: 'Installeer PicoClaw voordat u de runtime van PicoClaw start.',
        progressDescription: 'PicoClaw wordt gedownload en geïnstalleerd.',
        stages: {
          preparing: 'Voorbereiden',
          downloading: 'Downloaden',
          extracting: 'Uitpakken',
          verifying: 'Verifiëren',
          installing: 'Installeren',
          installed: 'Geïnstalleerd',
          install_timeout: 'Time-out',
          install_failed: 'Mislukt'
        }
      },
      model: {
        requiredTitle: 'Modelconfiguratie is vereist',
        requiredDescription: 'Configureer het PicoClaw-model voordat u PicoClaw chat gebruikt.',
        docsTitle: 'Configuratiehandleiding',
        docsDesc: 'Ondersteunde modellen en protocollen',
        menuLabel: 'Model configureren',
        modelIdentifier: 'Modelidentificatie',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API-sleutel',
        apiKeyPlaceholder: 'Voer de API-sleutel van het model in',
        save: 'Opslaan',
        saving: 'Opslaan',
        saved: 'Modelconfiguratie opgeslagen',
        saveFailed: 'Kan de modelconfiguratie niet opslaan',
        invalid: 'Model-ID, API Base URL en API-sleutel zijn vereist'
      },
      uninstall: {
        menuLabel: 'Verwijderen',
        confirmTitle: 'PicoClaw verwijderen',
        confirmContent:
          'Weet u zeker dat u PicoClaw wilt verwijderen? Hiermee worden het uitvoerbare bestand en alle configuratiebestanden verwijderd.',
        confirmOk: 'Verwijderen',
        confirmCancel: 'Annuleren'
      },
      history: {
        title: 'Geschiedenis',
        loading: 'Sessies laden...',
        emptyTitle: 'Nog geen geschiedenis',
        emptyDescription: 'Eerdere PicoClaw sessies verschijnen hier.',
        loadFailed: 'Kan de sessiegeschiedenis niet laden',
        deleteFailed: 'Kan sessie niet verwijderen',
        deleteConfirmTitle: 'Sessie verwijderen',
        deleteConfirmContent: 'Weet u zeker dat u "{{title}}" wilt verwijderen?',
        deleteConfirmOk: 'Verwijderen',
        deleteConfirmCancel: 'Annuleren',
        messageCount_one: '{{count}} bericht',
        messageCount_other: '{{count}} berichten',
        messageCount: '{{count}} berichten'
      },
      config: {
        startRuntime: 'Start PicoClaw',
        stopRuntime: 'Stop PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Bediening overschakelen naar PicoClaw?',
        enableConfirmDesc:
          'Bij het starten van PicoClaw wordt de externe MCP-service uitgeschakeld.',
        enableConfirmOk: 'PicoClaw starten',
        enableConfirmCancel: 'Annuleren',
        title: 'Start PicoClaw',
        description: 'Start de runtime om de PicoClaw assistent te gaan gebruiken.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Er is een probleem opgetreden',
      refresh: 'Vernieuwen'
    },
    fullscreen: {
      toggle: 'Volledig scherm schakelen'
    },
    menu: {
      collapse: 'Menu samenvouwen',
      expand: 'Menu uitvouwen'
    }
  }
};

export default nl;
