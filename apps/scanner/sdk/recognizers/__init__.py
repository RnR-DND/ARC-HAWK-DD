"""
Recognizers Package
===================
Custom Presidio recognizers with mathematical validation.
"""

from .aadhaar import AadhaarRecognizer
from .pan import PANRecognizer
from .credit_card import CreditCardRecognizer
from .email import EmailRecognizer
from .phone import IndianPhoneRecognizer
from .driving_license import DrivingLicenseRecognizer
from .passport import IndianPassportRecognizer
from .voter_id import VoterIDRecognizer
from .bank_account import BankAccountRecognizer
from .ifsc import IFSCRecognizer
from .upi import UPIRecognizer

__all__ = [
    'AadhaarRecognizer',
    'PANRecognizer',
    'CreditCardRecognizer',
    'EmailRecognizer',
    'IndianPhoneRecognizer',
    'DrivingLicenseRecognizer',
    'IndianPassportRecognizer',
    'VoterIDRecognizer',
    'BankAccountRecognizer',
    'IFSCRecognizer',
    'UPIRecognizer',
]
