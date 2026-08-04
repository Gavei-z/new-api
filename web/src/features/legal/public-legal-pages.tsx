/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  PRIVACY_RELATED_LINKS,
  PRIVACY_SECTIONS,
  REFUND_RELATED_LINKS,
  REFUND_SECTIONS,
  TERMS_RELATED_LINKS,
  TERMS_SECTIONS,
} from './public-legal-content'
import { PublicLegalDocument } from './public-legal-document'

export function TermsOfService() {
  return (
    <PublicLegalDocument
      titleKey='Terms of Service'
      descriptionKey='legal.terms.description'
      contactKey='legal.terms.contact'
      sections={TERMS_SECTIONS}
      relatedLinks={TERMS_RELATED_LINKS}
    />
  )
}

export function PublicPrivacyPolicy() {
  return (
    <PublicLegalDocument
      titleKey='Privacy Policy'
      descriptionKey='legal.privacy.description'
      contactKey='legal.privacy.contact'
      sections={PRIVACY_SECTIONS}
      relatedLinks={PRIVACY_RELATED_LINKS}
    />
  )
}

export function RefundPolicy() {
  return (
    <PublicLegalDocument
      titleKey='Refund Policy'
      descriptionKey='legal.refund.description'
      contactKey='legal.refund.contact'
      sections={REFUND_SECTIONS}
      relatedLinks={REFUND_RELATED_LINKS}
    />
  )
}
